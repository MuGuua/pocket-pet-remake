package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"pocket-pet-remake/server/internal/module/bag"
)

// 显式指定本地专用测试库时运行真实 SQL；每次使用独立临时 schema 并在结束时清理。
func TestRuntimeBagPagePostgres(t *testing.T) {
	db := openLocalRuntimeFixture(t, bagPageFixtureSQL)
	ctx := context.Background()
	// 包装后不暴露 BeginTx：只验证分页读取，不在测试中重复测试原堆叠规范化。
	baseline, err := NewBagRepository(&pageObservedDB{DBTX: db}).ListRuntimeContainer(ctx, 1, bag.ContainerTypeBag)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []bag.RuntimePageQuery{
		{}, {Page: 2, PageSize: 28}, {Category: "equipment", Page: 2, PageSize: 7},
		{Category: "other", Page: 1, PageSize: 5}, {Category: "enhance_material", Page: 3, PageSize: 7},
		{Category: " EQUIPMENT ", Page: 1, PageSize: 3}, {Category: "unknown"},
		{Page: 100, PageSize: 28}, {Page: ^uint32(0), PageSize: ^uint32(0)},
	} {
		t.Run(fmt.Sprintf("%+v", query), func(t *testing.T) {
			observed := &pageObservedDB{DBTX: db, pageOnly: true}
			actual, err := NewBagRepository(observed).ListRuntimeBagPage(ctx, 1, query)
			if err != nil {
				t.Fatal(err)
			}
			expected := query.Apply(*baseline)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("SQL page differs from full snapshot slice:\nactual=%+v\nexpected=%+v", actual, expected)
			}
			if observed.materialReads > 1 {
				t.Fatal("repeated material aggregation")
			}
			if len(actual.Snapshot.Items) == 0 && observed.itemReads != 0 {
				t.Fatal("empty page should skip item query")
			}
			if len(actual.Snapshot.Items) > 0 && observed.itemReads != 1 {
				t.Fatal("expected one LIMIT query")
			}
		})
	}
	empty, err := NewBagRepository(&pageObservedDB{DBTX: db, pageOnly: true}).ListRuntimeBagPage(ctx, 2, bag.RuntimePageQuery{})
	if err != nil || empty == nil || empty.Snapshot.UsedSlots != 0 || empty.TotalItems != 0 || len(empty.Snapshot.Items) != 0 {
		t.Fatalf("empty container: %+v %v", empty, err)
	}
	missing, err := NewBagRepository(&pageObservedDB{DBTX: db}).ListRuntimeBagPage(ctx, 999, bag.RuntimePageQuery{})
	if err != nil || missing != nil {
		t.Fatal("missing container contract changed")
	}
}

// 查询观察器不改数据库结果，只阻止分页路径退化到全量明细读取并统计预览材料查询。
type pageObservedDB struct {
	DBTX
	pageOnly      bool
	itemReads     int
	materialReads int
}

func (db *pageObservedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if query == runtimeContainerItemsQuery && db.pageOnly {
		return nil, fmt.Errorf("paged request performed a full item read")
	}
	if query == runtimeContainerPageItemsQuery {
		db.itemReads++
	}
	if query == runtimeBagEnhanceMaterialsQuery {
		db.materialReads++
	}
	return db.DBTX.QueryContext(ctx, query, args...)
}

// 仅供独立 schema 的测试结构与样本，不读取或更改正式游戏数据。
const bagPageFixtureSQL = `
CREATE TABLE player_container(player_id bigint,container_type text,capacity integer,max_capacity integer);
CREATE TABLE player_container_item(id bigint,player_id bigint,container_type text,slot_index integer,item_id bigint,item_uid text,quantity bigint,is_bound boolean,expire_at timestamptz,entered_at timestamptz);
CREATE TABLE item_definition(item_id bigint PRIMARY KEY,is_enabled boolean DEFAULT true,sort_weight integer DEFAULT 0,item_name text,item_type text,item_sub_type text,quality integer,icon text,required_level integer,usable boolean,can_drop boolean,can_sell boolean,sell_price_copper bigint,target_type text,effect_type text,"desc" text);
CREATE TABLE equipment_instance(item_uid text PRIMARY KEY,enhance_level integer,is_damaged boolean);
CREATE TABLE item_equipment_extra(item_id bigint PRIMARY KEY,equip_slot text,appearance_skin_id text,appearance_only boolean,base_hp integer,base_mana integer,base_atk integer,base_def integer,base_spd integer,base_stats_json jsonb,enhance_per_level_stats_json jsonb,can_enhance boolean,max_enhance_level integer);
INSERT INTO player_container VALUES(1,'bag',80,80),(2,'bag',80,80);
INSERT INTO item_definition(item_id,item_name,item_type,item_sub_type,"desc")
SELECT i,'item'||i,CASE WHEN i%3=0 THEN 'EQUIPMENT' ELSE 'consumable' END,CASE WHEN i%3=1 THEN 'equipment_enhance' ELSE 'misc' END,'' FROM generate_series(1,60) i;
INSERT INTO player_container_item
SELECT i,1,'bag',61-i,i,'',1,false,null,'2026-01-01'::timestamptz+(i%5)*interval '1 second' FROM generate_series(1,60) i;
`

// 显式专用本地测试库中创建隔离 schema，复用连接与清理规则，不读取正式数据库配置。
func openLocalRuntimeFixture(t *testing.T, fixtureSQL string) *sql.DB {
	t.Helper()
	dsn := os.Getenv("POCKET_BAG_PAGE_TEST_DSN")
	if dsn == "" {
		t.Skip("set POCKET_BAG_PAGE_TEST_DSN to a dedicated local pocket_bag_page_test database")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database != "pocket_bag_page_test" || !(cfg.Host == "127.0.0.1" || strings.HasPrefix(cfg.Host, "/tmp/")) {
		t.Fatal("requires a dedicated local pocket_bag_page_test database")
	}
	setup := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { setup.Close() })
	schema := fmt.Sprintf("bag_page_%d", time.Now().UnixNano())
	if _, err = setup.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := setup.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { db.Close() })
	if _, err = db.Exec(fixtureSQL); err != nil {
		t.Fatal(err)
	}
	return db
}
