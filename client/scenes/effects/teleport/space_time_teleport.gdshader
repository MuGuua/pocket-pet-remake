shader_type canvas_item;
render_mode blend_add, unshaded;

uniform int effect_mode = 0;
uniform int ring_layer = 0;
uniform float progress : hint_range(0.0, 1.0, 0.001) = 0.0;
uniform vec4 beam_color : source_color = vec4(0.08, 0.55, 1.0, 1.0);

float activation_envelope(float value) {
    float build = smoothstep(0.0, 0.34, value);
    float fade = 1.0 - smoothstep(0.70, 0.82, value);
    return build * fade;
}

vec4 render_beam(vec2 uv, float time_value) {
    float concentration = smoothstep(0.08, 0.68, progress);
    float center_offset = abs(uv.x - 0.5) * 2.0;
    float wave = sin(uv.y * 24.0 - time_value * 2.8) * 0.025;
    wave += sin(uv.y * 71.0 + time_value * 7.0) * 0.012;
    float beam_width = mix(0.95, 0.36, concentration);
    float body = 1.0 - smoothstep(beam_width * 0.18, beam_width, center_offset + wave);
    float core = 1.0 - smoothstep(0.0, beam_width * 0.22, center_offset);
    float vertical_fade = smoothstep(0.0, 0.13, uv.y) * (1.0 - smoothstep(0.94, 1.0, uv.y));
    float moving_bands = 0.72 + sin(uv.y * 96.0 - time_value * 9.0) * 0.18;
    float density = mix(0.24, 1.0, concentration);
    float alpha = body * vertical_fade * moving_bands * density * activation_envelope(progress);
    vec3 color = beam_color.rgb * (1.2 + core * mix(0.6, 2.4, concentration));
    return vec4(color, clamp(alpha * beam_color.a, 0.0, 1.0));
}

vec4 render_ring(vec2 uv, float time_value) {
    vec2 centered_uv = (uv - vec2(0.5)) * 2.0;
    float radius = length(centered_uv);
    float concentration = smoothstep(0.05, 0.68, progress);
    float ring_radius = mix(0.625, 0.50, concentration);
    float thickness = mix(0.19, 0.055, concentration);
    float ring = 1.0 - smoothstep(thickness, thickness + 0.08, abs(radius - ring_radius));
    float halo = 1.0 - smoothstep(thickness + 0.04, thickness + 0.30, abs(radius - ring_radius));
    float angle = atan(centered_uv.y, centered_uv.x);
    float arc_pulse = 0.72 + sin(angle * 14.0 - time_value * 6.0) * 0.28;
    float front_mask = smoothstep(0.48, 0.52, uv.y);
    float layer_mask = 1.0;
    if (ring_layer == 1) {
        layer_mask = 1.0 - front_mask;
    } else if (ring_layer == 2) {
        layer_mask = front_mask;
    }
    float alpha = (ring + halo * 0.34) * arc_pulse * activation_envelope(progress) * layer_mask;
    vec3 color = beam_color.rgb * (1.6 + ring * 1.8);
    return vec4(color, clamp(alpha * beam_color.a, 0.0, 1.0));
}

vec4 render_flash(vec2 uv) {
    vec2 centered_uv = uv * 2.0 - vec2(1.0);
    float radius = length(centered_uv);
    float peak = smoothstep(0.30, 0.58, progress) * (1.0 - smoothstep(0.62, 0.78, progress));
    float halo = pow(max(1.0 - radius, 0.0), 2.2);
    float core = 1.0 - smoothstep(0.0, 0.22, radius);
    float alpha = (halo * 0.62 + core) * peak;
    return vec4(beam_color.rgb * (1.8 + core * 2.2), clamp(alpha * beam_color.a, 0.0, 1.0));
}

vec4 render_cross_flash(vec2 uv) {
    vec2 centered_uv = abs(uv * 2.0 - vec2(1.0));
    float appear = smoothstep(0.70, 0.76, progress);
    float disappear = 1.0 - smoothstep(0.84, 1.0, progress);
    float collapse = smoothstep(0.78, 0.98, progress);
    float arm_length = mix(0.52, 0.12, collapse);
    float horizontal = exp(-pow(centered_uv.y * 14.0, 2.0))
        * exp(-pow(centered_uv.x / arm_length, 2.0));
    float vertical = exp(-pow(centered_uv.x * 14.0, 2.0))
        * exp(-pow(centered_uv.y / arm_length, 2.0));
    float radius = length(centered_uv);
    float core = exp(-radius * radius * 45.0);
    float halo = exp(-radius * radius * 7.0);
    float alpha = max(horizontal, vertical) + core * 0.85 + halo * 0.32;
    alpha *= appear * disappear;
    vec3 color = beam_color.rgb * (2.4 + core * 2.6);
    return vec4(color, clamp(alpha * beam_color.a, 0.0, 1.0));
}

vec4 render_demo_background(vec2 uv) {
    vec2 centered_uv = uv - vec2(0.5);
    float vignette = 1.0 - smoothstep(0.18, 0.78, length(centered_uv));
    vec2 grid_uv = fract(uv * vec2(13.0, 24.0));
    float grid_line = 1.0 - smoothstep(0.015, 0.055, min(grid_uv.x, grid_uv.y));
    vec3 base = mix(vec3(0.008, 0.018, 0.055), vec3(0.015, 0.070, 0.130), vignette);
    base += beam_color.rgb * grid_line * 0.035;
    return vec4(base, 1.0);
}

void fragment() {
    if (effect_mode == 1) {
        COLOR = render_ring(UV, TIME);
    } else if (effect_mode == 2) {
        COLOR = render_flash(UV);
    } else if (effect_mode == 3) {
        COLOR = render_demo_background(UV);
    } else if (effect_mode == 4) {
        COLOR = render_cross_flash(UV);
    } else {
        COLOR = render_beam(UV, TIME);
    }
}
