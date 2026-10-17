#include "aac.h"

#include <android/log.h>
#include <media/NdkMediaCodec.h>
#include <media/NdkMediaFormat.h>
#include <pthread.h>
#include <string.h>

#define TAG COUNTERTOP_PREFIX "-airplay"
#define logw(...) __android_log_print(ANDROID_LOG_WARN, TAG, __VA_ARGS__)
#define logi(...) __android_log_print(ANDROID_LOG_INFO, TAG, __VA_ARGS__)

#define CT_AAC_LC 4
#define CT_AAC_ELD 8
#define RATE 44100
#define CHANNELS 2

static pthread_mutex_t lock = PTHREAD_MUTEX_INITIALIZER;
static AMediaCodec *codec;
static int open_ct;
static pcm_sink out;
static uint32_t out_rate = RATE, out_channels = CHANNELS;

static void drain(void) {
    for (;;) {
        AMediaCodecBufferInfo info;
        ssize_t i = AMediaCodec_dequeueOutputBuffer(codec, &info, 0);
        if (i == AMEDIACODEC_INFO_OUTPUT_FORMAT_CHANGED) {
            AMediaFormat *f = AMediaCodec_getOutputFormat(codec);
            int32_t v;
            if (AMediaFormat_getInt32(f, AMEDIAFORMAT_KEY_SAMPLE_RATE, &v)) out_rate = (uint32_t)v;
            if (AMediaFormat_getInt32(f, AMEDIAFORMAT_KEY_CHANNEL_COUNT, &v)) out_channels = (uint32_t)v;
            AMediaFormat_delete(f);
            logi("audio decodes to %u Hz x%u", out_rate, out_channels);
            continue;
        }
        if (i == AMEDIACODEC_INFO_OUTPUT_BUFFERS_CHANGED) continue;
        if (i < 0) return;
        size_t size;
        uint8_t *b = AMediaCodec_getOutputBuffer(codec, (size_t)i, &size);
        if (b && info.size > 0) out(out_rate, out_channels, (const int16_t *)(b + info.offset), (uint32_t)info.size / 2);
        AMediaCodec_releaseOutputBuffer(codec, (size_t)i, false);
    }
}

static void drop(void) {
    if (!codec) return;
    AMediaCodec_stop(codec);
    AMediaCodec_delete(codec);
    codec = NULL;
}

static int start(void) {
    static const uint8_t eld[] = {0xf8, 0xe8, 0x50, 0x00};
    static const uint8_t lc[] = {0x12, 0x10};
    const uint8_t *csd;
    size_t csd_len;
    int32_t profile;
    switch (open_ct) {
    case CT_AAC_ELD:
        csd = eld, csd_len = sizeof(eld), profile = 39;
        break;
    case CT_AAC_LC:
        csd = lc, csd_len = sizeof(lc), profile = 2;
        break;
    default:
        return -1;
    }
    AMediaCodec *c = AMediaCodec_createDecoderByType("audio/mp4a-latm");
    if (!c) {
        logw("no AAC decoder");
        return -1;
    }
    AMediaFormat *f = AMediaFormat_new();
    AMediaFormat_setString(f, AMEDIAFORMAT_KEY_MIME, "audio/mp4a-latm");
    AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_SAMPLE_RATE, RATE);
    AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_CHANNEL_COUNT, CHANNELS);
    AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_AAC_PROFILE, profile);
    AMediaFormat_setInt32(f, AMEDIAFORMAT_KEY_IS_ADTS, 0);
    AMediaFormat_setBuffer(f, "csd-0", (void *)csd, csd_len);
    media_status_t st = AMediaCodec_configure(c, f, NULL, NULL, 0);
    AMediaFormat_delete(f);
    if (st == AMEDIA_OK) st = AMediaCodec_start(c);
    if (st != AMEDIA_OK) {
        AMediaCodec_delete(c);
        logw("AAC decoder for type %d failed: %d", open_ct, st);
        return -1;
    }
    codec = c;
    out_rate = RATE, out_channels = CHANNELS;
    logi("AAC decoder open for type %d", open_ct);
    return 0;
}

int aac_open(int ct, pcm_sink sink) {
    if (ct != CT_AAC_ELD && ct != CT_AAC_LC) {
        logw("no decoder for audio type %d", ct);
        return -1;
    }
    pthread_mutex_lock(&lock);
    if (open_ct != ct) drop();
    open_ct = ct;
    out = sink;
    pthread_mutex_unlock(&lock);
    return 0;
}

void aac_decode(const uint8_t *data, int len, uint32_t rtp_time) {
    pthread_mutex_lock(&lock);
    if (len <= 0 || (!codec && start() < 0)) {
        pthread_mutex_unlock(&lock);
        return;
    }
    ssize_t i = AMediaCodec_dequeueInputBuffer(codec, 20000);
    if (i >= 0) {
        size_t size;
        uint8_t *b = AMediaCodec_getInputBuffer(codec, (size_t)i, &size);
        size_t n = (size_t)len < size ? (size_t)len : size;
        if (b) memcpy(b, data, n);
        AMediaCodec_queueInputBuffer(codec, (size_t)i, 0, n, (uint64_t)rtp_time * 1000000u / RATE, 0);
    }
    drain();
    pthread_mutex_unlock(&lock);
}

void aac_flush(void) {
    pthread_mutex_lock(&lock);
    drop();
    pthread_mutex_unlock(&lock);
}
