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
	for (int i = 0; i < 5; i++) {
		v += a * vnoise(q);
		q = q * 2.03 + vec2(17.1, 9.7);
		a *= 0.5;
	}
	return v;
}

vec3 ramp4(vec3 a, vec3 b, vec3 c, vec3 d, float s) {
	s = clamp(s, 0.0, 1.0) * 3.0;
	if (s < 1.0) return mix(a, b, s);
	if (s < 2.0) return mix(b, c, s - 1.0);
	return mix(c, d, s - 2.0);
}

vec3 tone(vec3 c) { return vec3(1.0) - exp(-c * 1.35); }

void main() {
	highp float t = u[0];
	float s = clamp(u[1], 0.0, 1.0);
	highp float aspect = u[2];
	highp vec2 px = pixel();
	highp vec2 uv = px / res;
	highp vec2 p = vec2(uv.x * aspect, uv.y);
	highp vec2 mid = vec2(0.5 * aspect, 0.0);

	float rise = s * s * (3.0 - 2.0 * s);
	float sunY = mix(1.06, 0.5, rise);
	highp vec2 sun = vec2(0.5 * aspect, sunY);
	float rad = 0.07;
	float d = float(length(p - sun));
	float height = clamp(1.0 - float(uv.y), 0.0, 1.0);

	vec3 zenith = ramp4(vec3(0.004, 0.006, 0.03), vec3(0.04, 0.05, 0.16), vec3(0.14, 0.24, 0.5), vec3(0.24, 0.46, 0.8), s);
	vec3 low = ramp4(vec3(0.012, 0.01, 0.035), vec3(0.3, 0.08, 0.2), vec3(0.85, 0.36, 0.18), vec3(0.95, 0.72, 0.5), s);
	vec3 sky = mix(low, zenith, pow(height, 0.6));

	vec3 warm = mix(vec3(1.0, 0.36, 0.1), vec3(1.0, 0.8, 0.55), s);
	float lit = smoothstep(0.0, 0.5, s);
	float halo = lit * (1.0 - 0.7 * smoothstep(0.5, 1.0, s));
	sky += warm * 0.35 * exp(-d * 4.0) * halo;
	sky += warm * 0.5 * exp(-d * 14.0) * halo;
	float horizonGlow = exp(-abs(float(uv.y) - 0.88) * 9.0) * exp(-abs(float(p.x - sun.x)) * 1.2);
	sky += vec3(1.0, 0.42, 0.18) * horizonGlow * 0.45 * (1.0 - s) * lit;

	highp vec2 cell = floor(px / 4.0);
	float h = float(hash(cell));
	if (h > 0.996) {
		float tw = 0.55 + 0.45 * sin(float(t) * (1.2 + h * 3.0) + h * 50.0);
		sky += vec3(0.85, 0.9, 1.0) * tw * (h - 0.996) / 0.004 * (1.0 - smoothstep(0.05, 0.4, s)) * smoothstep(0.2, 0.6, height);
	}

	highp vec2 cq = vec2(float(p.x) * 1.3 + float(t) * 0.006, float(uv.y) * 5.0);
	float cl = fbm(cq * 1.6);
	float clouds = smoothstep(0.46, 0.72, cl) * smoothstep(0.12, 0.3, height) * smoothstep(0.92, 0.6, height);
	float under = clamp((float(uv.y) - sunY) * -1.2 + 0.6, 0.0, 1.0);
	vec3 dark = mix(vec3(0.015, 0.015, 0.04), vec3(0.3, 0.25, 0.38), s);
	vec3 lift = mix(vec3(0.5, 0.18, 0.2), mix(vec3(1.0, 0.6, 0.42), vec3(1.0, 0.96, 0.92), s), lit);
	vec3 cloud = mix(dark, lift, clamp(exp(-d * 2.2) * 1.2 * under + 0.35 * s, 0.0, 1.0));
	sky = mix(sky, cloud, clouds * 0.9);

	vec3 core = mix(vec3(1.0, 0.36, 0.08), vec3(1.0, 0.95, 0.82), smoothstep(0.05, 0.8, s));
	float disc = smoothstep(rad + 0.003, rad - 0.003, d);
	float limb = 1.0 - 0.3 * pow(clamp(d / rad, 0.0, 1.0), 2.0);
	sky = mix(sky, core * limb * 1.6, disc);
	float ang = atan(float(p.y - sun.y), float(p.x - sun.x));
	float rays = pow(0.5 + 0.5 * sin(ang * 11.0 + float(t) * 0.03), 3.0) * (0.6 + 0.4 * vnoise(vec2(ang * 6.0, float(t) * 0.1)));
	sky += core * rays * exp(-d * 5.5) * 0.07 * lit;

	float hill = 0.9 + 0.022 * fbm(vec2(float(p.x) * 1.8 + 3.0, 1.0)) - 0.03 * vnoise(vec2(float(p.x) * 5.0, 2.0));
	float far = 0.865 + 0.03 * fbm(vec2(float(p.x) * 1.2 + 9.0, 4.0));
	float near = smoothstep(hill - 0.0015, hill + 0.0015, float(uv.y));
	float farMask = smoothstep(far - 0.0015, far + 0.0015, float(uv.y)) * (1.0 - near);
	vec3 haze = mix(vec3(0.03, 0.02, 0.07), vec3(0.42, 0.3, 0.36), lit);
	sky = mix(sky, mix(sky, haze, 0.65), farMask);
	sky = mix(sky, mix(vec3(0.004, 0.003, 0.01), vec3(0.07, 0.05, 0.08), lit), near);

	vec3 col = tone(sky);

	float ink = texture2D(t0, vtex).a;
	highp vec2 nudge = vec2(4.0, 6.0) / res;
	float shadow = texture2D(t0, vtex - vec2(-nudge.x, nudge.y)).a;
	col = mix(col, col * 0.55, shadow * 0.6);
	col = mix(col, mix(vec3(0.82, 0.85, 0.98), vec3(1.0, 0.98, 0.95), s), ink * (0.55 + 0.45 * s));

	col *= 0.18 + 0.82 * smoothstep(0.0, 0.3, s);
#ifdef LIGHT
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
#else
	gl_FragColor = vec4(clamp(col + glow() * 0.6, 0.0, 1.0), 1.0);
#endif
}
