#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <vector>

#include "needle.h"

namespace whistle {
class Engine {
public:
    void set_depth(int);
};
}

extern whistle::Engine* g_whistle_engine asm("_ZN12_GLOBAL__N_19g_whistleE");

static std::vector<unsigned char> slurp(const char* path) {
    std::vector<unsigned char> b;
    FILE* f = fopen(path, "rb");
    if (!f) return b;
    fseek(f, 0, SEEK_END);
    b.resize(static_cast<size_t>(ftell(f)));
    fseek(f, 0, SEEK_SET);
    if (fread(b.data(), 1, b.size(), f) != b.size()) b.clear();
    fclose(f);
    return b;
}

static double fastest(const std::vector<float>& pcm, char* out, size_t cap) {
    double best = 0;
    for (int i = 0; i < 3; i++) {
        if (needle_transcribe(pcm.data(), static_cast<int>(pcm.size()), "en", nullptr, 0, out, static_cast<int>(cap)) <
            0) {
            fprintf(stderr, "transcribe: %s\n", needle_last_error());
            exit(1);
        }
        const char* p = strstr(out, "\"decode_tps\":");
        double tps = p ? atof(p + 13) : 0;
        if (tps > best) best = tps;
    }
    return best;
}

int main(int argc, char** argv) {
    if (argc < 3) {
        fprintf(stderr, "usage: depth_test MODEL SPEECH.raw\n");
        return 2;
    }
    auto cact = slurp(argv[1]);
    if (cact.empty() || needle_load(cact.data(), cact.size()) < 0) {
        fprintf(stderr, "load %s: %s\n", argv[1], needle_last_error());
        return 1;
    }
    auto raw = slurp(argv[2]);
    std::vector<float> pcm(raw.size() / 2);
    const int16_t* s = reinterpret_cast<const int16_t*>(raw.data());
    for (size_t i = 0; i < pcm.size(); i++) pcm[i] = s[i] / 32768.0f;

    char full_text[4096], shallow_text[4096];
    double full = fastest(pcm, full_text, sizeof full_text);
    g_whistle_engine->set_depth(2);
    double shallow = fastest(pcm, shallow_text, sizeof shallow_text);

    printf("full   %6.1f tok/s  %s\n", full, full_text);
    printf("depth2 %6.1f tok/s  %s\n", shallow, shallow_text);
    if (!strstr(shallow_text, "ndifference")) {
        fprintf(stderr, "FAIL: depth 2 transcript %s, want \"indifference\"\n", shallow_text);
        return 1;
    }
    if (shallow < full * 1.5) {
        fprintf(stderr, "FAIL: depth 2 %.1f tok/s, full %.1f tok/s, want at least 1.5x\n", shallow, full);
        return 1;
    }
    printf("depth: ok\n");
    return 0;
}
