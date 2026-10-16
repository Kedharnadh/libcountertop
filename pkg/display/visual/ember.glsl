vec3 heat(float h) {
	h = clamp(h, 0.0, 1.3);
	vec3 c = mix(vec3(0.0), vec3(0.35, 0.025, 0.004), smoothstep(0.0, 0.22, h));
	c = mix(c, vec3(0.9, 0.24, 0.03), smoothstep(0.18, 0.5, h));
	c = mix(c, vec3(1.0, 0.6, 0.2), smoothstep(0.45, 0.8, h));
	return mix(c, vec3(1.0, 0.9, 0.72), smoothstep(0.8, 1.25, h));
}

void main() {
	highp float t = u[0];
	float s = u[1];
	highp vec2 uv = pixel() / res;
	vec3 col = vec3(0.0);
	if (float(uv.y) > 0.56) {
		float wob = sin(float(uv.y) * 40.0 - float(t) * 1.6 + float(uv.x) * 9.0) * 0.006;
		vec3 bed = texture2D(t1, vec2(uv.x, (float(uv.y) - 0.56) / 0.44 + wob)).rgb;
		float flick = 0.6 + 0.4 * sin(float(t) * (0.6 + bed.g * 1.4) + bed.g * 30.0);
		float body = bed.r * 1.15 * (0.35 + 0.65 * bed.g * flick);
		col = heat(body * bed.b * u[5]);
	}
	float radiance = exp(-max(0.65 - float(uv.y), 0.0) * u[2]);
	col += vec3(1.0, 0.32, 0.06) * radiance * u[3];
	col += vec3(1.0, 0.5, 0.16) * u[4] * (1.0 - float(uv.y) * 0.6);

	col = vec3(1.0) - exp(-col * 1.25);

	float ink = texture2D(t0, vtex).a;
	highp vec2 nudge = vec2(4.0, 6.0) / res;
	float shadow = texture2D(t0, vtex - vec2(-nudge.x, nudge.y)).a;
	col = mix(col, col * 0.45, shadow * 0.6);
	col = mix(col, vec3(1.0, 0.86, 0.68), ink * (0.6 + 0.35 * s));

	col *= u[6];
#ifdef LIGHT
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
#else
	gl_FragColor = vec4(clamp(col + glow() * 0.7, 0.0, 1.0), 1.0);
#endif
}
