extends Node

# 宠物或编队数据刷新后向外广播当前宠物总数。
signal pets_updated(count: int)

# 处理宠物列表响应，并把宠物与编队摘要写入全局状态。
func handle_pet_list(payload: Dictionary) -> void:
	# 读取服务端返回的宠物列表载荷。
	var pets_variant: Variant = payload.get("pets", [])
	# 读取服务端返回的编队摘要载荷。
	var lineup_variant: Variant = payload.get("lineup", [])
	# 规范化宠物列表为数组结构。
	var pets: Array = pets_variant if pets_variant is Array else []
	# 规范化编队摘要为数组结构。
	var lineup: Array = lineup_variant if lineup_variant is Array else []
	GameState.set_pets(pets, lineup)
	pets_updated.emit(GameState.pets.size())

# 处理单只宠物更新推送，并把结果合并进全局状态。
func handle_pet_update(payload: Dictionary) -> void:
	# 兼容 pet 字段和直接宠物结构两种载荷格式。
	var pet_variant: Variant = payload.get("pet", payload)
	# 规范化单只宠物数据为字典结构。
	var pet: Dictionary = pet_variant if pet_variant is Dictionary else {}
	GameState.upsert_pet(pet)
	pets_updated.emit(GameState.pets.size())

# 处理宠物属性点分配响应，并把最新宠物快照写回全局状态。
func handle_allocate_attr_response(payload: Dictionary) -> void:
	var pet_variant: Variant = payload.get("pet", {})
	if pet_variant is Dictionary:
		GameState.upsert_pet(pet_variant)
	pets_updated.emit(GameState.pets.size())

# 处理宠物技能详情响应，合并完整 skill_slots 后写回全局状态。
func handle_pet_skill_detail_response(payload: Dictionary) -> void:
	var pet_variant: Variant = payload.get("pet", {})
	if pet_variant is Dictionary:
		GameState.upsert_pet(pet_variant)
	pets_updated.emit(GameState.pets.size())

# 处理法宝装备/卸下响应，刷新宠物快照。
func handle_pet_artifact_response(payload: Dictionary) -> void:
	var pet_variant: Variant = payload.get("pet", {})
	if pet_variant is Dictionary:
		GameState.upsert_pet(pet_variant)
	pets_updated.emit(GameState.pets.size())

# 处理编队设置响应，并在成功时刷新全局编队状态。
func handle_lineup_set_response(payload: Dictionary) -> void:
	if not bool(payload.get("accepted", false)):
		return
	# 读取服务端返回的最新编队摘要。
	var lineup_variant: Variant = payload.get("lineup", [])
	# 规范化编队摘要为数组结构。
	var lineup: Array = lineup_variant if lineup_variant is Array else []
	GameState.set_lineup(lineup)
	pets_updated.emit(GameState.pets.size())
