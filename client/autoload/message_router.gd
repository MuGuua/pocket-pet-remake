extends Node

var _handlers: Dictionary = {}

func register_handler(cmd: int, handler: Callable) -> void:
    if not _handlers.has(cmd):
        _handlers[cmd] = []

    var handlers: Array = _handlers[cmd]
    if handlers.has(handler):
        return

    handlers.append(handler)
    _handlers[cmd] = handlers

func unregister_handler(cmd: int, handler: Callable) -> void:
    if not _handlers.has(cmd):
        return

    var handlers: Array = _handlers[cmd]
    handlers.erase(handler)
    if handlers.is_empty():
        _handlers.erase(cmd)
    else:
        _handlers[cmd] = handlers

func route_message(cmd: int, payload: Dictionary = {}) -> void:
    if not _handlers.has(cmd):
        push_warning("No handler registered for cmd %d" % cmd)
        return

    var handlers: Array = _handlers[cmd]
    for handler_variant in handlers:
        if handler_variant is Callable and handler_variant.is_valid():
            handler_variant.call(payload)
