#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include <string>
#include <vector>

#include "needle.h"
#include "protocol.h"

namespace whistle {
class Engine {
public:
    void set_depth(int);
};
}

extern whistle::Engine* g_whistle_engine asm("_ZN12_GLOBAL__N_19g_whistleE");

namespace {

struct state {
    int fd;
    std::string keywords;
    std::vector<float> pcm;
    char out[8192];
};

bool write_all(int fd, const void* data, size_t length) {
    const char* p = static_cast<const char*>(data);
    while (length > 0) {
        ssize_t n = write(fd, p, length);
        if (n < 0 && errno == EINTR) continue;
        if (n <= 0) return false;
        p += n;
        length -= static_cast<size_t>(n);
    }
    return true;
}

bool read_all(int fd, void* data, size_t length) {
    char* p = static_cast<char*>(data);
    while (length > 0) {
        ssize_t n = read(fd, p, length);
        if (n < 0 && errno == EINTR) continue;
        if (n <= 0) return false;
        p += n;
        length -= static_cast<size_t>(n);
    }
    return true;
}

bool send(int fd, uint32_t kind, const void* head, size_t head_length, const char* text, size_t text_length) {
    voice_header h{kind, static_cast<uint32_t>(head_length + text_length)};
    return write_all(fd, &h, sizeof h) && write_all(fd, head, head_length) && write_all(fd, text, text_length);
}

void failed(int fd, const char* what) {
    fprintf(stderr, "%s\n", what);
    send(fd, MSG_FAILED, nullptr, 0, what, strlen(what));
}

double now_ms() {
    timespec t;
    clock_gettime(CLOCK_MONOTONIC, &t);
    return t.tv_sec * 1000.0 + t.tv_nsec / 1e6;
}

double field(const char* json, const char* key) {
    const char* p = strstr(json, key);
    return p ? atof(p + strlen(key)) : 0;
}

void transcribe(state* s, const int16_t* samples, size_t length, uint32_t start_ms, uint32_t end_ms) {
    s->pcm.resize(length);
    for (size_t i = 0; i < length; i++) s->pcm[i] = samples[i] / 32768.0f;
    const char* keywords = s->keywords.empty() ? nullptr : s->keywords.c_str();
    double began = now_ms();
    if (needle_transcribe(s->pcm.data(), static_cast<int>(length), "en", keywords, 1, s->out, sizeof s->out) < 0) {
        failed(s->fd, needle_last_error());
        return;
    }
    voice_text t{start_ms, end_ms, static_cast<uint32_t>(now_ms() - began),
                 static_cast<uint32_t>(field(s->out, "\"decode_tps\":") * 100)};
    send(s->fd, MSG_TEXT, &t, sizeof t, s->out, strlen(s->out));
}

bool load(const char* path) {
    FILE* f = fopen(path, "rb");
    if (!f) return false;
    std::vector<unsigned char> cact;
    if (fseek(f, 0, SEEK_END) == 0) {
        long n = ftell(f);
        if (n > 0) {
            cact.resize(static_cast<size_t>(n));
            fseek(f, 0, SEEK_SET);
            if (fread(cact.data(), 1, cact.size(), f) != cact.size()) cact.clear();
        }
    }
    fclose(f);
    return !cact.empty() && needle_load(cact.data(), cact.size()) >= 0;
}

}

int main(int argc, char** argv) {
    static state s;
    s.fd = VOICE_FD;
    if (argc < 3) {
        failed(s.fd, "usage: voice MODEL DEPTH");
        return 2;
    }
    int depth = atoi(argv[2]);
    if (!load(argv[1])) {
        failed(s.fd, needle_last_error());
        return 1;
    }
    if (depth > 0) g_whistle_engine->set_depth(depth);
    voice_ready r{VOICE_VERSION, static_cast<uint32_t>(depth)};
    if (!send(s.fd, MSG_READY, &r, sizeof r, nullptr, 0)) return 1;

    std::vector<char> payload;
    for (;;) {
        voice_header h;
        if (!read_all(s.fd, &h, sizeof h)) return 0;
        if (h.length > VOICE_MAX_PAYLOAD) {
            failed(s.fd, "payload too large");
            return 1;
        }
        payload.resize(h.length);
        if (h.length && !read_all(s.fd, payload.data(), h.length)) return 0;
        switch (h.kind) {
        case MSG_CLIP: {
            if (h.length < sizeof(voice_clip)) break;
            voice_clip c;
            memcpy(&c, payload.data(), sizeof c);
            size_t count = (h.length - sizeof c) / 2;
            std::vector<int16_t> samples(count);
            memcpy(samples.data(), payload.data() + sizeof c, count * 2);
            transcribe(&s, samples.data(), count, c.start_ms, c.end_ms);
            break;
        }
        case MSG_KEYWORDS:
            s.keywords.assign(payload.data(), h.length);
            break;
        default:
            break;
        }
    }
}
