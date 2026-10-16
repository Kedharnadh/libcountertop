highp float hash(highp vec2 q) {
	highp vec3 p3 = fract(vec3(q.xyx) * 0.1031);
	p3 += dot(p3, p3.yzx + 33.33);
	return fract((p3.x + p3.y) * p3.z);
}

float vnoise(highp vec2 q) {
	highp vec2 i = floor(q);
	vec2 f = vec2(fract(q));
	f = f * f * (3.0 - 2.0 * f);
	float a = float(hash(i)), b = float(hash(i + vec2(1.0, 0.0)));
	float c = float(hash(i + vec2(0.0, 1.0))), d = float(hash(i + vec2(1.0, 1.0)));
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float fbm(highp vec2 q) {
	float v = 0.0, a = 0.5;
	for (int i = 0; i < 4; i++) {
		v += a * vnoise(q);
		q = q * 2.02 + vec2(5.3, 11.7);
		a *= 0.5;
	}
	return v;
}

vec3 coals(highp vec2 q, highp float t) {
	highp vec2 i = floor(q);
	highp vec2 f = fract(q);
	float near = 8.0, next = 8.0;
	highp vec2 owner = i;
	for (int y = -1; y <= 1; y++) {
		for (int x = -1; x <= 1; x++) {
			highp vec2 g = vec2(float(x), float(y));
			highp vec2 o = vec2(hash(i + g + 3.1), hash(i + g + 19.7));
			float dd = float(length(g + o - f));
			if (dd < near) {
				next = near;
				near = dd;
				owner = i + g;
			} else if (dd < next) {
				next = dd;
			}
		}
	}
	float seed = float(hash(owner + 7.7));
	float flick = 0.6 + 0.4 * sin(float(t) * (0.6 + seed * 1.4) + seed * 30.0);
	return vec3(near, next - near, seed * flick);
}

vec3 heat(float h) {
	h = clamp(h, 0.0, 1.3);
	vec3 c = mix(vec3(0.0), vec3(0.35, 0.025, 0.004), smoothstep(0.0, 0.22, h));
	c = mix(c, vec3(0.9, 0.24, 0.03), smoothstep(0.18, 0.5, h));
	c = mix(c, vec3(1.0, 0.6, 0.2), smoothstep(0.45, 0.8, h));
	return mix(c, vec3(1.0, 0.9, 0.72), smoothstep(0.8, 1.25, h));
}

void main() {
	highp float t = u[0];
	float s = clamp(u[1], 0.0, 1.0);
	highp float aspect = u[2];
	highp vec2 px = pixel();
	highp vec2 uv = px / res;
	highp vec2 p = vec2(uv.x * aspect, uv.y);
	float lift = s * s * (3.0 - 2.0 * s);

	float bedTop = 0.6;
	float depth = smoothstep(bedTop - 0.04, bedTop + 0.12, float(uv.y));
	float shimmer = (fbm(vec2(float(p.x) * 4.0, float(uv.y) * 3.0 - float(t) * 0.25)) - 0.5) * 0.012;
	float near = max(float(uv.y) + shimmer - bedTop, 0.0) / (1.0 - bedTop) + 0.12;
	highp vec2 q = vec2((float(p.x) - 0.5 * float(aspect)) / near * 3.2, 2.6 / near);
	vec3 c = coals(q, t);
	float crust = fbm(q * 2.3 + c.z * 9.0);
	float inner = smoothstep(0.62, 0.05, c.x);
	float gap = smoothstep(0.02, 0.12, c.y);
	float body = inner * gap * (0.35 + 0.65 * c.z) * (0.55 + 0.6 * smoothstep(0.35, 0.75, crust));
	float h = body * depth * mix(0.3, 1.3, lift);

	float radiance = exp(-max(float(bedTop) + 0.05 - float(uv.y), 0.0) * mix(9.0, 2.2, lift));
	vec3 air = vec3(1.0, 0.32, 0.06) * radiance * mix(0.05, 0.45, lift);
	air += vec3(1.0, 0.5, 0.16) * lift * lift * 0.22 * (1.0 - float(uv.y) * 0.6);

	vec3 col = heat(h) + air;

	float sparks = 0.0;
	for (int k = 0; k < 3; k++) {
		float fk = float(k);
		highp vec2 g = vec2(float(p.x) * (16.0 + fk * 8.0), float(uv.y) * (9.0 + fk * 5.0) + float(t) * (0.35 + fk * 0.2));
		highp vec2 id = floor(g);
		highp vec2 f = fract(g) - 0.5;
		float r = float(hash(id + fk * 31.0));
		if (r > 0.94) {
			highp vec2 off = vec2(sin(float(t) * 1.1 + r * 40.0) * 0.28, 0.0);
			float dd = float(length(f - off));
			float fade = smoothstep(0.1, bedTop, float(uv.y)) * (0.3 + 0.7 * lift);
			sparks += smoothstep(0.1, 0.0, dd) * fade * (r - 0.94) / 0.06;
		}
	}
	col += vec3(1.0, 0.62, 0.25) * sparks;

	col = vec3(1.0) - exp(-col * 1.25);

	float ink = texture2D(t0, vtex).a;
	highp vec2 nudge = vec2(4.0, 6.0) / res;
	float shadow = texture2D(t0, vtex - vec2(-nudge.x, nudge.y)).a;
	col = mix(col, col * 0.45, shadow * 0.6);
	col = mix(col, vec3(1.0, 0.86, 0.68), ink * (0.6 + 0.35 * s));

	col *= 0.3 + 0.7 * smoothstep(0.0, 0.25, s);
#ifdef LIGHT
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
#else
	gl_FragColor = vec4(clamp(col + glow() * 0.7, 0.0, 1.0), 1.0);
#endif
}
