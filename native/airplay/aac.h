#pragma once
#include <stdint.h>

typedef void (*pcm_sink)(uint32_t rate, uint32_t channels, const int16_t *pcm, uint32_t samples);

int aac_open(int ct, pcm_sink sink);
void aac_decode(const uint8_t *data, int len, uint32_t rtp_time);
void aac_flush(void);
