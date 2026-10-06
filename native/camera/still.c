#include "platform.h"
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include <android/log.h>
#include <media/NdkMediaCodec.h>
#include <media/NdkMediaFormat.h>

#include "still.h"

#define TAG COUNTERTOP_PREFIX "-camera"
#define logw(...) __android_log_print(ANDROID_LOG_WARN, TAG, __VA_ARGS__)

#define STEP_US 20000
#define TRIES 50
#define YUV420_PLANAR 19
#define YUV420_SEMIPLANAR 21

static unsigned char clamp(int v) { return v < 0 ? 0 : v > 255 ? 255 : (unsigned char)v; }

static unsigned char *to_rgba(const uint8_t *buf, size_t n, AMediaFormat *f, int w, int h) {
	int32_t color = 0, stride = w, rows = h, left = 0, top = 0;
	AMediaFormat_getInt32(f, AMEDIAFORMAT_KEY_COLOR_FORMAT, &color);
	AMediaFormat_getInt32(f, "stride", &stride);
	AMediaFormat_getInt32(f, "slice-height", &rows);
	AMediaFormat_getInt32(f, "crop-left", &left);
	AMediaFormat_getInt32(f, "crop-top", &top);
	if (stride < w) stride = w;
	if (rows < h) rows = h;
	if (color != YUV420_PLANAR && color != YUV420_SEMIPLANAR) {
		logw("still: decoder colour format %#x", color);
		return NULL;
	}
	size_t luma = (size_t)stride * rows;
	if (n < luma + luma / 2) {
		logw("still: decoded frame is %zu bytes for %dx%d", n, stride, rows);
		return NULL;
	}
	const uint8_t *y = buf, *u, *v;
	int cs, step;
	if (color == YUV420_PLANAR) {
		u = buf + luma;
		v = u + luma / 4;
		cs = stride / 2;
		step = 1;
	} else {
		u = buf + luma;
		v = u + 1;
		cs = stride;
		step = 2;
	}
	unsigned char *out = malloc((size_t)w * h * 4);
	for (int r = 0; r < h; r++) {
		int sr = r + top;
		for (int c = 0; c < w; c++) {
			int sc = c + left;
			int Y = y[sr * stride + sc] - 16;
			int U = u[(sr / 2) * cs + (sc / 2) * step] - 128;
			int V = v[(sr / 2) * cs + (sc / 2) * step] - 128;
			unsigned char *p = out + ((size_t)r * w + c) * 4;
			p[0] = clamp((298 * Y + 409 * V + 128) >> 8);
			p[1] = clamp((298 * Y - 100 * U - 208 * V + 128) >> 8);
			p[2] = clamp((298 * Y + 516 * U + 128) >> 8);
			p[3] = 255;
		}
	}
	return out;
}

unsigned char *still_decode(const uint8_t *csd, size_t csdn, const uint8_t *key, size_t keyn, int w, int h) {
	AMediaCodec *dec = AMediaCodec_createDecoderByType("video/avc");
	if (!dec) return NULL;
	unsigned char *out = NULL;

	AMediaFormat *f = AMediaFormat_new();
	AMediaFormat_setString(f, AMEDIAFORMAT_KEY_MIME, "video/avc");
	AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_WIDTH, w);
	AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_HEIGHT, h);
	media_status_t st = AMediaCodec_configure(dec, f, NULL, NULL, 0);
	AMediaFormat_delete(f);
	if (st != AMEDIA_OK || AMediaCodec_start(dec) != AMEDIA_OK) {
		logw("still: decoder refused %dx%d: %d", w, h, st);
		AMediaCodec_delete(dec);
		return NULL;
	}

	int fed = 0, ended = 0;
	for (int t = 0; t < TRIES && !out && !ended; t++) {
		if (fed < 2) {
			ssize_t in = AMediaCodec_dequeueInputBuffer(dec, STEP_US);
			if (in >= 0) {
				size_t cap = 0;
				uint8_t *buf = AMediaCodec_getInputBuffer(dec, in, &cap);
				if (fed == 0 && buf && csdn + keyn <= cap) {
					memcpy(buf, csd, csdn);
					memcpy(buf + csdn, key, keyn);
					AMediaCodec_queueInputBuffer(dec, in, 0, csdn + keyn, 0, 0);
				} else {
					AMediaCodec_queueInputBuffer(dec, in, 0, 0, 0, AMEDIACODEC_BUFFER_FLAG_END_OF_STREAM);
				}
				fed++;
			}
		}
		AMediaCodecBufferInfo info;
		ssize_t o = AMediaCodec_dequeueOutputBuffer(dec, &info, STEP_US);
		if (o < 0) continue;
		if (info.flags & AMEDIACODEC_BUFFER_FLAG_END_OF_STREAM) ended = 1;
		if (info.size > 0) {
			size_t cap = 0;
			uint8_t *buf = AMediaCodec_getOutputBuffer(dec, o, &cap);
			AMediaFormat *of = AMediaCodec_getOutputFormat(dec);
			if (buf && of) out = to_rgba(buf + info.offset, (size_t)info.size, of, w, h);
			if (!out) ended = 1;
			if (of) AMediaFormat_delete(of);
		}
		AMediaCodec_releaseOutputBuffer(dec, o, false);
	}
	if (!out && !ended) logw("still: the decoder gave no picture");

	AMediaCodec_stop(dec);
	AMediaCodec_delete(dec);
	return out;
}
