#pragma once
#include <stdint.h>

#define VOICE_VERSION 2
#define VOICE_FD 3
#define VOICE_RATE 16000
#define VOICE_MAX_PAYLOAD (1 << 20)

#define MSG_CLIP 'C'
#define MSG_KEYWORDS 'K'
#define MSG_READY 'R'
#define MSG_TEXT 'T'
#define MSG_FAILED 'F'

struct voice_header {
    uint32_t kind;
    uint32_t length;
};

struct voice_ready {
    uint32_t version;
    uint32_t depth;
};

struct voice_clip {
    uint32_t start_ms;
    uint32_t end_ms;
};

struct voice_text {
    uint32_t start_ms;
    uint32_t end_ms;
    uint32_t took_ms;
    uint32_t decode_tps_x100;
};
