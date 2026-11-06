package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pocket-pet-remake/server/internal/module/pet"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// 详情禁止写库和全量宠物/法宝/战斗编队查询，仍使用真实 SQL 与数据库返回数据。
type petDetailObservedDB struct {
	DBTX
	singleReads int
	writes      int
}

func (db *petDetailObservedDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	if q == listPetsByPlayerIDQuery || q == listArtifactEquipmentByPlayerIDQuery || q == listLineupByPlayerIDQuery {
		return nil, fmt.Errorf("detail unexpectedly read all pet combat data")
	}
	if q == findPetByUIDQuery {
		db.singleReads++
	}
	return db.DBTX.QueryContext(ctx, q, args...)
}
func (db *petDetailObservedDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	db.writes++
	return nil, fmt.Errorf("detail unexpectedly wrote a snapshot")
}

// 同一只宠物的单只 SQL 与原全量计算一致；源属性/法宝变更后立刻可见，不依赖写视图。
func TestPetDetailReadOnlyPostgres(t *testing.T) {
	db := openLocalRuntimeFixture(t, petDetailFixtureSQL())
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	baseline, err := NewPetRepository(db).ListPetsByPlayerID(ctx, 1)
	if err != nil || len(baseline) != 2 {
		t.Fatalf("baseline: %v %+v", err, baseline)
	}
	observed := &petDetailObservedDB{DBTX: db}
	repo := NewPetRepository(observed)
	single, err := repo.FindPetByUID(ctx, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	expected := baseline[0]
	expected.InLineup = true
	if !reflect.DeepEqual(single, expected) {
		t.Fatalf("single detail differs from full read:\nactual=%+v\nexpected=%+v", single, expected)
	}
	svc := pet.NewService(repo, nil, nil, nil)
	first, err := svc.GetPetDetail(ctx, 1, 11)
	if err != nil || !first.InLineup || !first.IsUsable || first.CritRatePct != 20 {
		t.Fatalf("detail rules changed: %+v %v", first, err)
	}
	if _, err = db.Exec("UPDATE player_pet SET hp=1,crit_rate_pct=9,custom_name='updated' WHERE player_id=1 AND id=11; UPDATE pet_artifact_equipment SET skill_id=2002 WHERE player_id=1 AND pet_uid=11"); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.GetPetDetail(ctx, 1, 11)
	if err != nil || fresh.HP != 1 || fresh.CritRatePct != 9 || fresh.CustomName != "updated" {
		t.Fatalf("stale source data: %+v %v", fresh, err)
	}
	found := false
	for _, id := range fresh.SkillIDs {
		if id == 2002 {
			found = true
		}
		if id == 1001 {
			t.Fatal("stale artifact skill")
		}
	}
	if !found {
		t.Fatal("fresh artifact skill missing")
	}
	if _, err = svc.GetPetDetail(ctx, 2, 11); !errors.Is(err, pet.ErrPetNotFound) {
		t.Fatalf("ownership check: %v", err)
	}
	if observed.writes != 0 || observed.singleReads != 4 {
		t.Fatalf("unexpected detail work: %+v", observed)
	}
}

// 按现有查询的原始字段建立隔离测试结构；只用来构造扫描类型，不生成正式迁移。
func petDetailFixtureSQL() string {
	seen := map[string]bool{"player_id": true}
	for _, m := range regexp.MustCompile(`pp\.([a-z_0-9]+)`).FindAllStringSubmatch(listPetsByPlayerIDSelectQuery, -1) {
		seen[m[1]] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	columns := make([]string, 0, len(names))
	for _, name := range names {
		kind := "bigint DEFAULT 0"
		if strings.HasSuffix(name, "_enabled") {
			kind = "boolean DEFAULT false"
		}
		if strings.HasSuffix(name, "skill_ids") {
			kind = "jsonb DEFAULT '[]'::jsonb"
		}
		if name == "custom_name" || name == "grant_source" {
			kind = "text DEFAULT ''"
		}
		columns = append(columns, name+" "+kind)
	}
	return "CREATE TABLE player_pet (" + strings.Join(columns, ",") + ");" + `
CREATE TABLE pet_definition(pet_id bigint PRIMARY KEY,pet_name text,skin_id text,aptitude_profile text,status integer);
CREATE TABLE player_lineup(player_id bigint,pet_uid bigint,slot_index integer);
CREATE TABLE pet_artifact_equipment(player_id bigint,pet_uid bigint,slot_index integer,skill_id bigint);
CREATE TABLE pet_combat_stat_cap(stat_key text,cap_value bigint,status integer);
INSERT INTO pet_definition VALUES(1,'pet','skin','normal',1);
INSERT INTO player_pet(id,player_id,pet_id,level,hp,hp_max,crit_rate_pct) VALUES(11,1,1,10,100,200,99),(12,1,1,10,100,200,99),(21,2,1,10,100,200,99);
INSERT INTO player_lineup VALUES(1,11,0),(2,21,0);
INSERT INTO pet_artifact_equipment VALUES(1,11,0,1001),(1,12,0,1002),(2,21,0,1003);
INSERT INTO pet_combat_stat_cap VALUES('crit_rate_pct',20,1);
`
}
