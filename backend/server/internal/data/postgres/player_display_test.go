package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pocket-pet-remake/server/internal/module/player"
	"reflect"
	"strings"
	"testing"
)

// 只读展示禁止写视图；计数验证一次源记录、一次加点规则与一次装备查询。
type playerDisplayObservedDB struct {
	DBTX
	writes, queries int
	failRules       bool
}

func (db *playerDisplayObservedDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	db.writes++
	return nil, fmt.Errorf("display attempted snapshot write")
}
func (db *playerDisplayObservedDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	db.queries++
	return db.DBTX.QueryRowContext(ctx, q, args...)
}
func (db *playerDisplayObservedDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	db.queries++
	if db.failRules && q == listPlayerCombatSnapshotAttrRulesQuery {
		return nil, fmt.Errorf("rules unavailable")
	}
	return db.DBTX.QueryContext(ctx, q, args...)
}

// 与真实写入并重新读取的旧快照逐字段一致，同时反映属性、配置与装备变化。
func TestPlayerDisplayReadOnlyPostgres(t *testing.T) {
	db := openLocalRuntimeFixture(t, playerDisplayFixtureSQL())
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	legacy := NewPlayerRepository(db)
	observed := &playerDisplayObservedDB{DBTX: db}
	svc := player.NewService(NewPlayerRepository(observed), nil, nil, nil)
	for i, update := range []string{"", "UPDATE player SET strength=5,gold=456,hp=999,name='changed' WHERE id=1; UPDATE equipment_instance SET is_damaged=true", "UPDATE player_attr_convert_config SET convert_rate=3 WHERE source_attr='strength'; UPDATE equipment_instance SET is_damaged=false", "UPDATE player_equipment_slot SET equip_slot='costume'; UPDATE item_equipment_extra SET appearance_only=true,appearance_skin_id='new-skin'", `UPDATE player_equipment_slot SET equip_slot='weapon'; UPDATE item_equipment_extra SET appearance_only=false,enhance_per_level_stats_json='{"atk":2}'; UPDATE equipment_instance SET enhance_level=2`} {
		if update != "" {
			if _, err := db.Exec(update); err != nil {
				t.Fatal(err)
			}
		}
		before := observed.queries
		actual, err := svc.GetDisplayProfile(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if observed.writes != 0 || observed.queries-before != 3 {
			t.Fatalf("display queries=%d writes=%d", observed.queries-before, observed.writes)
		}
		if err := legacy.RefreshPlayerCombatSnapshot(ctx, 1); err != nil {
			t.Fatal(err)
		}
		expected, err := legacy.FindPlayerCombatSnapshot(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("case %d display differs from persisted view:\nactual=%+v\nexpected=%+v", i, actual, expected)
		}
		if i == 3 && actual.SkinID != "new-skin" {
			t.Fatal("fresh costume appearance missing")
		}
		if i == 0 && (actual.ATK != 21 || actual.HP != 116) {
			t.Fatalf("equipment/allocation/HP clamp rules changed: %+v", actual)
		}
	}
	if _, err := svc.GetDisplayProfile(ctx, 999); !errors.Is(err, player.ErrPlayerNotFound) {
		t.Fatalf("missing player: %v", err)
	}
	observed.failRules = true
	if _, err := svc.GetDisplayProfile(ctx, 1); err == nil || observed.writes != 0 {
		t.Fatal("query failure must not fall back to a writable path")
	}
}

// 根据既有查询列建立隔离测试结构，包含全部原始与视图字段以检测遗漏，不生成正式迁移。
func playerDisplayFixtureSQL() string {
	source := findPlayerByIDQuery[strings.Index(findPlayerByIDQuery, "SELECT")+6 : strings.Index(findPlayerByIDQuery, "FROM player")]
	rawColumns := []string{"status bigint DEFAULT 1"}
	snapshotColumns := []string{}
	kind := func(name string) string {
		if name == "name" || name == "skin_id" {
			return "text DEFAULT ''"
		}
		if name == "skill_ids" {
			return "jsonb DEFAULT '[]'::jsonb"
		}
		if name == "updated_at" {
			return "timestamptz DEFAULT CURRENT_TIMESTAMP"
		}
		return "bigint DEFAULT 0"
	}
	for _, name := range strings.Split(source, ",") {
		name = strings.TrimSpace(name)
		suffix := ""
		if name == "id" {
			suffix = " PRIMARY KEY"
		}
		rawColumns = append(rawColumns, name+" "+kind(name)+suffix)
	}
	cols := upsertPlayerCombatSnapshotQuery[strings.Index(upsertPlayerCombatSnapshotQuery, "(")+1 : strings.Index(upsertPlayerCombatSnapshotQuery, ") VALUES")]
	for _, name := range strings.Split(cols, ",") {
		name = strings.TrimSpace(name)
		suffix := ""
		if name == "player_id" {
			suffix = " PRIMARY KEY"
		}
		snapshotColumns = append(snapshotColumns, name+" "+kind(name)+suffix)
	}
	return "CREATE TABLE player (" + strings.Join(rawColumns, ",") + "); CREATE TABLE player_combat_snapshot (" + strings.Join(snapshotColumns, ",") + ");" + `
CREATE TABLE player_attr_convert_config(id serial,source_attr text,target_attr text,convert_rate bigint,status bigint);
CREATE TABLE player_equipment_slot(player_id bigint,equip_slot text,item_uid text);
CREATE TABLE equipment_instance(item_uid text,item_id bigint,enhance_level bigint,is_damaged boolean);
CREATE TABLE item_definition(item_id bigint,item_name text,icon text,required_level bigint,"desc" text);
CREATE TABLE item_equipment_extra(item_id bigint,appearance_skin_id text,appearance_only boolean,base_hp bigint,base_mana bigint,base_atk bigint,base_def bigint,base_spd bigint,base_stats_json jsonb,enhance_per_level_stats_json jsonb,weapon_skills_json jsonb,enhance_per_level_weapon_skill_levels_json jsonb,weapon_type text);
INSERT INTO player(id,name,level,exp,strength,vitality,agility,mind,gold,hp,vigor,vigor_max,base_hp_max,base_atk,base_def,base_spd,base_mana,base_hit_pct,base_dodge_pct,skill_ids,skin_id,mercenary_resist_pct,generic_shield_pct,guard,talent_dmg_pct,talent_reduce_pct,element_adv_pct,element_penalty_pct,position_version)
VALUES(1,'player',10,99,2,3,4,5,123,999,60,100,100,10,20,30,40,95,3,'[1001,1101]','skin',11,12,13,14,15,16,17,42);
INSERT INTO player_attr_convert_config(source_attr,target_attr,convert_rate,status) VALUES('strength','atk',2,1),('vitality','hp_max',2,1),('mind','mana',2,1),('agility','spd',2,1);
INSERT INTO player_equipment_slot VALUES(1,'weapon','weapon-1');
INSERT INTO equipment_instance VALUES('weapon-1',101,0,false);
INSERT INTO item_definition VALUES(101,'weapon','',1,'');
INSERT INTO item_equipment_extra VALUES(101,'',false,10,2,7,3,4,'{}','{}','[]','{}','sword');
`
}
