extends Node

# 按消息号索引全部处理回调的注册表。
var _handlers: Dictionary = {}

# 为指定消息号注册一个处理回调。
func register_handler(cmd: int, handler: Callable) -> void:
	if not _handlers.has(cmd):
		_handlers[cmd] = []

	# 读取当前消息号下已经注册的回调列表。
	var handlers: Array = _handlers[cmd]
	if handlers.has(handler):
		return

	handlers.append(handler)
	_handlers[cmd] = handlers

# 为指定消息号注销一个处理回调。
func unregister_handler(cmd: int, handler: Callable) -> void:
	if not _handlers.has(cmd):
		return

	# 读取当前消息号下已经注册的回调列表。
	var handlers: Array = _handlers[cmd]
	handlers.erase(handler)
	if handlers.is_empty():
		_handlers.erase(cmd)
	else:
		_handlers[cmd] = handlers


# 查询指定消息号是否已有业务处理器；请求响应可据此避免无意义告警。
func has_handler(cmd: int) -> bool:
	return _handlers.has(cmd)


# 把消息载荷分发给指定消息号下的全部有效回调。
func route_message(cmd: int, payload: Dictionary = {}) -> void:
	if not _handlers.has(cmd):
		push_warning("No handler registered for cmd %d" % cmd)
		return

	# 读取当前消息号对应的回调列表。
	var handlers: Array = _handlers[cmd]
	for handler_variant in handlers:
		if handler_variant is Callable and handler_variant.is_valid():
			handler_variant.call(payload)
