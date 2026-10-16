vec3 at(int i) { return vec3(u[i], u[i + 1], u[i + 2]); }

void main() {
	float s = u[1];
	highp float aspect = u[2];
	float sunY = u[3];
	highp vec2 uv = pixel() / res;
	highp vec2 p = vec2(uv.x * aspect, uv.y);
	float d = float(length(p - vec2(0.5 * aspect, sunY)));
	float height = 1.0 - float(uv.y);

	vec3 sky = mix(at(7), at(10), sqrt(height));
	float near = max(1.0 - d * 1.6, 0.0);
	float tight = max(1.0 - d * 5.0, 0.0);
	sky += at(13) * (0.35 * near * near + 0.5 * tight * tight * tight) * u[4];
	float band = max(1.0 - abs(float(uv.y) - 0.88) * 6.0, 0.0);
	sky += vec3(1.0, 0.42, 0.18) * band * band * max(1.0 - abs(float(p.x) - 0.5 * float(aspect)) * 0.8, 0.0) * u[5];

	if (height > 0.12 && height < 0.92) {
		float cl = texture2D(t1, vec2(fract(float(p.x) * 0.26 + u[0]), float(uv.y))).r;
		float clouds = smoothstep(0.46, 0.72, cl) * smoothstep(0.12, 0.3, height) * smoothstep(0.92, 0.6, height);
		float under = clamp((float(uv.y) - sunY) * -1.2 + 0.6, 0.0, 1.0);
		vec3 cloud = mix(at(16), at(19), clamp(near * 1.2 * under + 0.35 * s, 0.0, 1.0));
		sky = mix(sky, cloud, clouds * 0.9);
	}

	float k = min(d / 0.07, 1.0);
	sky = mix(sky, at(22) * (1.0 - 0.3 * k * k) * 1.6, smoothstep(0.073, 0.067, d));

	if (float(uv.y) > 0.84) {
		vec3 ridge = texture2D(t2, vec2(float(p.x) / 2.0, 0.5)).rgb;
		float hill = 0.9 + 0.022 * ridge.r - 0.03 * ridge.g;
		float far = 0.865 + 0.03 * ridge.b;
		float front = smoothstep(hill - 0.0015, hill + 0.0015, float(uv.y));
		float back = smoothstep(far - 0.0015, far + 0.0015, float(uv.y)) * (1.0 - front);
		sky = mix(sky, mix(sky, at(25), 0.65), back);
		sky = mix(sky, at(28), front);
	}

	vec3 col = vec3(1.0) - exp(-sky * 1.35);

	float ink = texture2D(t0, vtex).a;
	highp vec2 nudge = vec2(4.0, 6.0) / res;
	float shadow = texture2D(t0, vtex - vec2(-nudge.x, nudge.y)).a;
	col = mix(col, col * 0.55, shadow * 0.6);
	col = mix(col, mix(vec3(0.82, 0.85, 0.98), vec3(1.0, 0.98, 0.95), s), ink * (0.55 + 0.45 * s));

	col *= u[6];
#ifdef LIGHT
	gl_FragColor = vec4(clamp(col, 0.0, 1.0), 1.0);
#else
	gl_FragColor = vec4(clamp(col + glow() * 0.6, 0.0, 1.0), 1.0);
#endif
}
