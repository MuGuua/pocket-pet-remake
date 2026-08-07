shader_type canvas_item;

// 受击闪白强度：1.0 为完全变白，0.0 为保持精灵原始颜色。
uniform float flash_amount : hint_range(0.0, 1.0, 0.01) = 0.0;

void fragment() {
    // 保留纹理透明度与节点 modulate，只将可见像素的 RGB 逐渐混合到白色。
    vec4 sampled_color = texture(TEXTURE, UV) * COLOR;
    vec3 flash_color = mix(sampled_color.rgb, vec3(1.0), flash_amount);
    COLOR = vec4(flash_color, sampled_color.a);
}
