highp float hash(highp vec2 q) {
	highp vec3 p3 = fract(vec3(q.xyx) * 0.1031);
	p3 += dot(p3, p3.yzx + 33.33);
	return fract((p3.x + p3.y) * p3.z);
}

void main() {
	highp float t = u[0];
	float s = clamp(u[1], 0.0, 1.0);
	highp float aspect = u[2];
	highp vec2 px = pixel();
	highp vec2 uv = px / res;
	highp vec2 p = vec2((uv.x - 0.5) * aspect, uv.y - 0.5);
	float r = float(length(p));

	float corner = float(length(vec2(0.5 * aspect, 0.5)));
	float grow = s * s * (3.0 - 2.0 * s);
	float edge = mix(0.04, corner * 1.25, grow);
	float soft = mix(0.1, corner * 0.6, grow);
	float body = smoothstep(edge + soft, edge - soft * 0.2, r);
	float core = exp(-r * r / (0.02 + 0.3 * grow));
	float breathe = 1.0 + 0.03 * sin(float(t) * 0.8) * (1.0 - grow);
	float level = clamp((body * mix(0.55, 1.0, grow) + core * 0.6) * breathe, 0.0, 1.0);
	level = mix(level, 1.0, smoothstep(0.8, 1.0, s));
	vec3 tint = mix(vec3(1.0, 0.9, 0.78), vec3(1.0), smoothstep(0.2, 0.9, s));
	float dither = (float(hash(px + fract(float(t) * 7.0))) - 0.5) / 255.0;
	vec3 col = tint * level + dither;
	float ink = texture2D(t0, vtex).a;
	vec3 face = mix(vec3(0.75, 0.7, 0.62), vec3(0.42, 0.42, 0.45), smoothstep(0.3, 0.9, level));
	col = mix(col, face, ink * (0.35 + 0.55 * s));
#ifdef LIGHT
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
#else
	gl_FragColor = vec4(clamp(col + glow() * (1.0 - s) * 0.5, 0.0, 1.0), 1.0);
#endif
}
