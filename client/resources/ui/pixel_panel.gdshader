shader_type canvas_item;
render_mode unshaded;

// 只替换旧菜单贴图的半透明中心，边框与透明转角保持原始像素颜色。
// 子节点不继承该材质，因此正文和图标不会随面板底色变淡。
uniform vec4 background_color : source_color = vec4(0.012, 0.016, 0.012, 0.68);
// UI 图集面板使用不透明深蓝中心；只替换接近该底色的像素，保留金色边框。
uniform bool atlas_background = false;
// 原描边向内扩展两个贴图像素，手机缩放后仍能辨认亮边与暗边。
uniform float border_inset_pixels : hint_range(0.0, 3.0, 1.0) = 2.0;

// 保留控件自身的显隐渐变；不把已采样的底图颜色再乘一次。
varying vec4 item_modulate;

void vertex() {
    item_modulate = COLOR;
}

void fragment() {
    vec4 original = texture(TEXTURE, UV);
    vec4 result = original;
    bool atlas_fill = atlas_background && original.a > 0.99 && distance(original.rgb, vec3(13.0, 28.0, 61.0) / 255.0) < 0.035;
    if (atlas_fill || (!atlas_background && original.a > 0.9 && original.a < 0.99)) {
        result = background_color;
        // 只有九宫格边缘需要额外采样，大片正文背景保留一次纹理读取。
        vec2 distance_to_edge = min(UV, vec2(1.0) - UV);
        if (!atlas_background && border_inset_pixels > 0.0 && (distance_to_edge.x < TEXTURE_PIXEL_SIZE.x * 8.0 || distance_to_edge.y < TEXTURE_PIXEL_SIZE.y * 8.0)) {
            vec2 step_size = TEXTURE_PIXEL_SIZE * border_inset_pixels;
            vec4 left_edge = texture(TEXTURE, UV - vec2(step_size.x, 0.0));
            vec4 right_edge = texture(TEXTURE, UV + vec2(step_size.x, 0.0));
            vec4 top_edge = texture(TEXTURE, UV - vec2(0.0, step_size.y));
            vec4 bottom_edge = texture(TEXTURE, UV + vec2(0.0, step_size.y));
            if (left_edge.a > 0.999) {
                result = left_edge;
            } else if (right_edge.a > 0.999) {
                result = right_edge;
            } else if (top_edge.a > 0.999) {
                result = top_edge;
            } else if (bottom_edge.a > 0.999) {
                result = bottom_edge;
            }
        }
    }
    COLOR = result * item_modulate;
}
