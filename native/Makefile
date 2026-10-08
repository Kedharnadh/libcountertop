NATIVE := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
NDK ?= $(firstword $(ANDROID_NDK_HOME) $(ANDROID_NDK_LATEST_HOME) $(wildcard /opt/homebrew/Caskroom/android-ndk/*/AndroidNDK*.app/Contents/NDK))
NDK_HOST := $(if $(filter Darwin,$(shell uname -s)),darwin-x86_64,linux-x86_64)
PREFIX ?= countertop
OUT ?= $(CURDIR)/bin
NDK_CC := $(NDK)/toolchains/llvm/prebuilt/$(NDK_HOST)/bin/armv7a-linux-androideabi$(API)-clang
SURFACE_BIN ?= $(OUT)/$(PREFIX)-surface
CAMERA_BIN ?= $(OUT)/$(PREFIX)-camera
CAMSHIM_BIN ?= $(OUT)/lib$(PREFIX)-camshim.so
STUBS := $(OUT)/countertop-stubs/$(PREFIX)-api$(API)
FLAGS := -O2 -Wall -Werror -I$(NATIVE) -DCOUNTERTOP_PREFIX='"$(PREFIX)"'
SURFACE := $(NATIVE)/surface
CAMERA := $(NATIVE)/camera
CAMERA_LIBS := camera_client gui utils binder $(if $(filter 25,$(API)),stagefright)
CAMERA_GPU_LIBS := $(if $(filter 27,$(API)),-lEGL -lGLESv2)
CAMERA_SOURCES := $(CAMERA)/camera.c $(CAMERA)/turn.c $(CAMERA)/cam2.c $(if $(filter 25,$(API)),$(CAMERA)/still.c)

.PHONY: all surface camera camshim check-ndk test
all: surface camera camshim
check-ndk:
	@test "$(API)" = 25 -o "$(API)" = 27 || { echo "supported native API levels: 25, 27"; exit 1; }
	@test -x "$(NDK_CC)" || { echo "no NDK clang at $(NDK_CC); set NDK or ANDROID_NDK_HOME"; exit 1; }

surface: check-ndk
	@mkdir -p $(dir $(SURFACE_BIN)) $(STUBS)/surface
	@for lib in gui utils binder; do \
		up=$$(echo $$lib | tr a-z A-Z); \
		$(NDK_CC) -shared -DLIB$$up -Wl,-soname,lib$$lib.so -o $(STUBS)/surface/lib$$lib.so $(SURFACE)/stubs.c || exit 1; \
	done
	$(NDK_CC) $(FLAGS) -o $(SURFACE_BIN) $(SURFACE)/surface.c $(SURFACE)/video.c $(SURFACE)/gl.c $(SURFACE)/ui.c $(SURFACE)/drm.c $(SURFACE)/audio.c $(SURFACE)/wire.c \
		-L$(STUBS)/surface -lgui -lutils -lbinder -landroid -lmediandk -lEGL -lGLESv2 -llog -lm -Wl,--allow-shlib-undefined

camera: check-ndk
	@mkdir -p $(dir $(CAMERA_BIN)) $(STUBS)/camera
	@for lib in $(CAMERA_LIBS); do \
		up=$$(echo $$lib | tr a-z A-Z); \
		$(NDK_CC) -shared -DLIB$$up -Wl,-soname,lib$$lib.so -o $(STUBS)/camera/lib$$lib.so $(CAMERA)/stubs.c || exit 1; \
	done
	$(NDK_CC) $(FLAGS) -o $(CAMERA_BIN) $(CAMERA_SOURCES) \
		-L$(STUBS)/camera $(addprefix -l,$(CAMERA_LIBS)) -lcamera2ndk -lmediandk -landroid $(CAMERA_GPU_LIBS) -llog -lm -Wl,--allow-shlib-undefined

camshim: check-ndk
	@mkdir -p $(dir $(CAMSHIM_BIN))
	$(NDK_CC) $(FLAGS) -Wno-unused-parameter -shared -fPIC -o $(CAMSHIM_BIN) $(NATIVE)/camshim/camshim.c

test:
	@mkdir -p $(OUT)
	$(or $(HOST_CC),cc) -O1 -g -Wall -Werror -fsanitize=undefined -fno-sanitize-recover=all -I$(NATIVE) -I$(SURFACE) -o $(OUT)/wire_test $(SURFACE)/wire.c $(SURFACE)/test/wire_test.c
	$(OUT)/wire_test
