BEGIN;

-- 以下位图由 client/tools/export_scene_navigation.gd 基于正式地图和玩家碰撞体生成。
INSERT INTO world_scene_navigation (
    scene_id, version, origin_x_milli, origin_y_milli, grid_width, grid_height,
    cell_size_milli, navigation_data, data_hash, walkable_cell_count, source_scene_path,
    status, change_reason, publish_reason
) VALUES
    (1, 1, 0, 0, 11, 15, 1000, decode('ffe000000003f87f0f8060380700f81f03c0080008', 'hex'), 'cccb6d77f2ac6ffb9b5f93fb1b4bd5946c774f2a52b3d2f9679e372227b495d9', 54, 'res://scenes/maps/fashtown/roxus_house.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (2, 1, 0, -4000, 11, 13, 1000, decode('fffffffffff002004008213ceffdfcfe1e02', 'hex'), '3cd9909ff01cf1a4ede5a8580dbb2dac98f71494404618c86ecbbef96cb34a58', 85, 'res://scenes/maps/fashtown/east_road_of_shanguang_town.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (3, 1, 0, 0, 14, 14, 1000, decode('fffc3003e00f8e0ff87c00f007c43fd0fcc3f32ffcfff8fc00', 'hex'), 'e4f2855bb8ee46f9874da266349e5f772c8a5ecea8bfdda5df9828c45f54461d', 108, 'res://scenes/maps/fashtown/radiant_market.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (4, 1, 0, 0, 9, 11, 1000, decode('ff80000203b0d0f8fc7e3f1f80', 'hex'), '69787cab8844d56514fe9da7b72999c519d2dc614e13c44bc58c905cb5564443', 47, 'res://scenes/maps/fashtown/bei_lu.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (5, 1, 0, 0, 13, 13, 1000, decode('fff80000000021fb0ff81800c21600f001840c216f00', 'hex'), '92d162c5fa796f9bcac14f91444ee78303b3134a7623defb7f05c0b8e2aaf3dd', 56, 'res://scenes/maps/fashtown/xue_xiao.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (6, 1, 0, 0, 10, 12, 1000, decode('ffc010e4791e4791fdf97e479007ff', 'hex'), '9e37379ce658eaa15f7377dbe9c3b217bcf8560138668587a6b3eaa8b8271ed5', 67, 'res://scenes/maps/fashtown/da_guai_qu.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (7, 1, 0, 0, 10, 12, 1000, decode('00c0300d837cdf3f0cc33cc03fffff', 'hex'), '172275e95c7509769b3f5140c41d73394268cf47962e74c5035bacf3d1a4f98a', 62, 'res://scenes/maps/fashtown/时光小屋.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (8, 1, 1000, 1000, 11, 14, 1000, decode('00200400801003f07fef3dffbfe7fcffdf8bf040', 'hex'), '40000ec32bfcb4f3b4ab363387c00ed03e99fe16ff185ea2bec75621f1db0606', 80, 'res://scenes/maps/fashtown/闪光镇传送区.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (9, 1, 1000, 1000, 24, 14, 1000, decode('0000010040010040010070fdf27ff9307ff9fffffdfffffdfffffdfffff91ffff907fff903fff903e031', 'hex'), 'feddc7e9322576de815d74bb4b1c21f9d8470416ff3f6453e2f6e83d94513277', 199, 'res://scenes/maps/闪光平原/闪耀广场.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (10, 1, 0, 0, 11, 13, 1000, decode('000000000003f80f01e03c078ff1f03e07e0', 'hex'), '05cf0b5a5c667bab87d17752c47ce8dcd37ae9aa53d7fd7a9f7e9985c01a3fed', 47, 'res://scenes/maps/闪光平原/闪光平原宠物学校.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (11, 1, 0, 0, 11, 13, 1000, decode('00000000000000003ff7feffdffbff7feffc', 'hex'), '487323a216bbfdb9e4a17a924394a9b53ef6cc54729c36acfb2291c2a599dc14', 70, 'res://scenes/maps/闪光平原/冰雪梦境.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (12, 1, 0, 0, 11, 13, 1000, decode('000000000000000e01c7bac24e4bff7fe3fc', 'hex'), 'ab2318075f477a6f1fc6c099b99ee24a15c4e21e769e90a90a53f7589ead30df', 51, 'res://scenes/maps/闪光平原/灰烬梦境.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (13, 1, 0, 0, 13, 13, 1000, decode('000000000020e3ff183bc1de0efff783201d00600300', 'hex'), 'a5af89f5ab10eca2ebd431236ef7ab6b2618af999b1523f57152ceec48aeb09e', 60, 'res://scenes/maps/闪光平原/翡翠梦境.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (14, 1, 0, 0, 11, 13, 1000, decode('000000000001c03807e3fc7f8ff1fe3fc7e0', 'hex'), 'ff4ff0c1d5fe1386d5ebbc2e862d9ab19fa9d32a7917e2b77f561bc8980cf536', 58, 'res://scenes/maps/闪光平原/阿尔的房间.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (15, 1, 0, 0, 11, 18, 1000, decode('000000000007f8ff1fe3fcffc180300607f8ff1fe3fc7f87e0', 'hex'), '7b0a49305ae52ea8236676817d05607cee552b38769d07971335838f914e4cf9', 94, 'res://scenes/maps/闪光平原/办公区.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (16, 1, 0, 0, 15, 13, 1000, decode('0000000000000003ff0fff00e0018003000603fff7ffefffc0', 'hex'), '68cae08eb73cfc0bb13be5b223d30635f65b7d60656b4c67adfea9aa9b0f4c8d', 73, 'res://scenes/maps/闪光平原/商业区.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (17, 1, 0, 0, 11, 13, 1000, decode('000000000000703f27e7feffd3f07e0fc1f8', 'hex'), 'bf0eff1bf68a48c0880ca14c6848e87886887a9c2468c48b5de0489a5b3bdc2b', 61, 'res://scenes/maps/闪光平原/报名区.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (18, 1, 0, 0, 11, 13, 1000, decode('000000000607f8ff1fe7fcff8ff1fe3fc3f0', 'hex'), '74c37bd840fca1b7b5f5b764029a4e05b98f828835e82fb1d8ed1a96a4488212', 74, 'res://scenes/maps/闪光平原/准备区.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (19, 1, 0, 0, 15, 20, 1000, decode('0000000000000004000f1c1e383c7878f0ffe1ffc3ffe7ffc1fe03fc07f80fe01fc3ff808000', 'hex'), 'fe4f96e3ac0d4f9b42c1ead621980add6a20af94ee8b9579744bcb17bb182182', 129, 'res://scenes/maps/闪光平原/家族会馆.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (20, 1, 0, 0, 13, 14, 1000, decode('000006007c03e07f03f07f83fe1ff0ff86403f00ff1ff8', 'hex'), '67a1e8f8bfeef9bb6500b469d34f5290c6b5f8f84c8c535c104523431b8ad713', 87, 'res://scenes/maps/闪光平原/闪光南路.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (21, 1, 0, 0, 18, 13, 1000, decode('00000000000001fffcffffbfffeffff9fffe1fff80ffe000180002000000', 'hex'), '2abf7449dc7fbb227738015e445dec0f6e475033dec6665d5c80d2d43c662869', 110, 'res://scenes/maps/闪光平原/五彩胡.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (22, 1, 0, 0, 11, 13, 1000, decode('0000000001e3e04f89f1fe31c7c8790fe0c0', 'hex'), '0663e3693d208a2e70209aca8498052457dcb55a37e3baaf3b30f0d529ce75f7', 54, 'res://scenes/maps/闪光平原/沼泽地.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (23, 1, 0, 0, 13, 14, 1000, decode('0007ffbffdffefff7ffbffdffefff7ffbff9ffc1f00000', 'hex'), '78783357d276baad717d4d71f74125471a598e1fa065de2daef372deae274126', 135, 'res://scenes/maps/闪光平原/闪光海岸.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (24, 1, 0, 0, 11, 13, 1000, decode('000000300600c03c0fc1f83f07e078060000', 'hex'), '71f45fbc59b0621aae1379bdc96f448d7f2c597ba121ad515821dee4851750e1', 40, 'res://scenes/maps/闪光平原/尘泥之地.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (25, 1, 0, 0, 15, 13, 1000, decode('0000000000000007878f0f1ffe3ffcfff9fff1ffe3ffc1fe00', 'hex'), '4b1aa622d32148f675175d47e5197e84e8940495b66517b3b9a327709dd5e394', 98, 'res://scenes/maps/闪光平原/精灵大厅.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布'),
    (26, 1, 0, 0, 12, 14, 1000, decode('0000600600601f03fc7fc60c60c60e60e7fc3fc1fc', 'hex'), '68f710c71b05dc4aac8f0f5046e7f7a77f3aa212b441f3233e560e51a01dfd71', 70, 'res://scenes/maps/闪光平原/海道.tscn', 1, 'P0-06 根据正式 Godot 地图资源初始化静态通行位图', 'P0-06 首批静态通行数据随迁移发布')
ON CONFLICT (scene_id, version) DO NOTHING;

-- 所有启用场景必须存在一个已发布导航版本，否则迁移失败，避免服务启动后普通移动全部被拒绝。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM world_scene_definition AS scene
        WHERE scene.status = 1
          AND NOT EXISTS (
              SELECT 1
              FROM world_scene_navigation AS navigation
              WHERE navigation.scene_id = scene.scene_id
                AND navigation.status = 1
          )
    ) THEN
        RAISE EXCEPTION 'enabled world scene is missing published navigation data';
    END IF;
END
$$;

COMMIT;
