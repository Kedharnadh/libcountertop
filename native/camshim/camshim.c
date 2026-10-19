#include <dlfcn.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>

#define WINDOW_FORMAT 2
#define WINDOW_CONSUMER_USAGE 10
#define FORMAT_RGBA_8888 1
#define FORMAT_IMPLEMENTATION_DEFINED 0x22
#define USAGE_VIDEO_ENCODER 0x10000

int check_permission(const void *perm, int pid, unsigned uid) __asm__("_ZN7android15checkPermissionERKNS_8String16Eij");
int check_permission(const void *perm, int pid, unsigned uid) { return 1; }

int check_calling(const void *perm) __asm__("_ZN7android22checkCallingPermissionERKNS_8String16E");
int check_calling(const void *perm) { return 1; }

int op_check(void *self, int op, int uid, const void *pkg) __asm__("_ZN7android13AppOpsManager7checkOpEiiRKNS_8String16E");
int op_check(void *self, int op, int uid, const void *pkg) { return 0; }

int op_note(void *self, int op, int uid, const void *pkg) __asm__("_ZN7android13AppOpsManager6noteOpEiiRKNS_8String16E");
int op_note(void *self, int op, int uid, const void *pkg) { return 0; }

int op_start(void *self, int op, int uid, const void *pkg) __asm__("_ZN7android13AppOpsManager7startOpEiiRKNS_8String16E");
int op_start(void *self, int op, int uid, const void *pkg) { return 0; }

void op_finish(void *self, int op, int uid, const void *pkg) __asm__("_ZN7android13AppOpsManager8finishOpEiiRKNS_8String16E");
void op_finish(void *self, int op, int uid, const void *pkg) {}

void op_watch(void *self, int op, const void *pkg, const void *cb)
	__asm__("_ZN7android13AppOpsManager17startWatchingModeEiRKNS_8String16ERKNS_2spINS_15IAppOpsCallbackEEE");
void op_watch(void *self, int op, const void *pkg, const void *cb) {}

void op_unwatch(void *self, const void *cb) __asm__("_ZN7android13AppOpsManager16stopWatchingModeERKNS_2spINS_15IAppOpsCallbackEEE");
void op_unwatch(void *self, const void *cb) {}

void sensors_service(void **sret, const void *name, int retry)
	__asm__("_ZN7android10frameworks13sensorservice4V1_014ISensorManager10getServiceERKNSt3__112basic_stringIcNS4_11char_traitsIcEENS4_9allocatorIcEEEEb");
void sensors_service(void **sret, const void *name, int retry) { *sret = NULL; }

void sound_load(void *self) __asm__("_ZN7android13CameraService9loadSoundEv");
void sound_load(void *self) {}

void sound_release(void *self) __asm__("_ZN7android13CameraService12releaseSoundEv");
void sound_release(void *self) {}

void sound_play(void *self, int kind) __asm__("_ZN7android13CameraService9playSoundENS0_10sound_kindE");
void sound_play(void *self, int kind) {}

int proc_scores(void *self, size_t n, int32_t *pids, int32_t *states, int32_t *scores)
	__asm__("_ZN7android18ProcessInfoService26getProcessStatesScoresImplEjPiS1_S1_");
int proc_scores(void *self, size_t n, int32_t *pids, int32_t *states, int32_t *scores) {
	for (size_t i = 0; i < n; i++) {
		states[i] = 2;
		scores[i] = 0;
	}
	return 0;
}

int request_priority(int pid, int tid, int prio, int app, int async) __asm__("_ZN7android15requestPriorityEiiibb");
int request_priority(int pid, int tid, int prio, int app, int async) { return 0; }

#if __ANDROID_API__ < 26
// Android 7's forms of the two above, as cronos's libcameraservice imports them.
int proc_states(void *self, size_t n, int32_t *pids, int32_t *states)
	__asm__("_ZN7android18ProcessInfoService20getProcessStatesImplEjPiS1_");
int proc_states(void *self, size_t n, int32_t *pids, int32_t *states) {
	for (size_t i = 0; i < n; i++) states[i] = 2;
	return 0;
}

int request_priority_n(int pid, int tid, int prio, int async) __asm__("_ZN7android15requestPriorityEiiib");
int request_priority_n(int pid, int tid, int prio, int async) { return 0; }

// Android 7's SensorManager waits for sensorservice, which lives in the stopped framework; NO_INIT is no sensors.
int sensors_state(void *self) __asm__("_ZN7android13SensorManager17assertStateLockedEv");
int sensors_state(void *self) { return -19; }

// MediaTek's EIS faults on a listener built over the missing sensor connection, and on a NULL one.
static int listener_ok(void) { return 1; }
static void *listener_vtable[32] = {[0 ... 31] = (void *)listener_ok};
static struct {
	void **vtable;
} quiet_listener = {listener_vtable};

void *sensor_listener(void) __asm__("_ZN14SensorListener14createInstanceEv");
void *sensor_listener(void) { return &quiet_listener; }

#endif

#if __ANDROID_API__ >= 28
// Android 9's CameraService::UidPolicy waits for the activity service, which lives in the stopped framework.
void am_observe(void *self, const void *observer, int event, int cutpoint, const void *pkg)
	__asm__("_ZN7android15ActivityManager19registerUidObserverERKNS_2spINS_12IUidObserverEEEiiRKNS_8String16E");
void am_observe(void *self, const void *observer, int event, int cutpoint, const void *pkg) {}

void am_unobserve(void *self, const void *observer)
	__asm__("_ZN7android15ActivityManager21unregisterUidObserverERKNS_2spINS_12IUidObserverEEE");
void am_unobserve(void *self, const void *observer) {}

int am_link(void *self, const void *recipient)
	__asm__("_ZN7android15ActivityManager11linkToDeathERKNS_2spINS_7IBinder14DeathRecipientEEE");
int am_link(void *self, const void *recipient) { return -32; }

int am_unlink(void *self, const void *recipient)
	__asm__("_ZN7android15ActivityManager13unlinkToDeathERKNS_2spINS_7IBinder14DeathRecipientEEE");
int am_unlink(void *self, const void *recipient) { return 0; }

int am_active(void *self, unsigned uid, const void *pkg) __asm__("_ZN7android15ActivityManager11isUidActiveEjRKNS_8String16E");
int am_active(void *self, unsigned uid, const void *pkg) { return 1; }

int package_uid(void *self, const void *pkg, int flags) __asm__("_ZN7android20PermissionController13getPackageUidERKNS_8String16Ei");
int package_uid(void *self, const void *pkg, int flags) { return -1; }

int op_start_quiet(void *self, int op, int uid, const void *pkg, int start_if_default)
	__asm__("_ZN7android13AppOpsManager14startOpNoThrowEiiRKNS_8String16Eb");
int op_start_quiet(void *self, int op, int uid, const void *pkg, int start_if_default) { return 0; }

int check_calling_ids(const void *perm, int *pid, int *uid) __asm__("_ZN7android22checkCallingPermissionERKNS_8String16EPiS3_");
int check_calling_ids(const void *perm, int *pid, int *uid) { return 1; }

#define CONNECT_DEVICE                                                                                                \
	"_ZN7android13CameraService13connectDeviceERKNS_2spINS_8hardware7camera222ICameraDeviceCallbacksEEERKNS_8String16" \
	"ESA_iPNS1_INS3_17ICameraDeviceUserEEE"

typedef void (*connect_device_fn)(void *status, void *self, const void *callbacks, const void *id, const void *pkg,
	int uid, void *device);

void s16_ctor(void *self, const char *s) __asm__("_ZN7android8String16C1EPKc");
void s16_dtor(void *self) __asm__("_ZN7android8String16D1Ev");

// Android 9's BasicClient waits 5 s for the permission service whenever the package name is empty, as NDK opens send it.
void connect_device(void *status, void *self, const void *callbacks, const void *id, const void *pkg, int uid,
	void *device) __asm__(CONNECT_DEVICE);
void connect_device(void *status, void *self, const void *callbacks, const void *id, const void *pkg, int uid,
	void *device) {
	static connect_device_fn real;
	if (!real) real = (connect_device_fn)dlsym(RTLD_NEXT, CONNECT_DEVICE);
	if (**(const uint16_t *const *)pkg) {
		real(status, self, callbacks, id, pkg, uid, device);
		return;
	}
	void *name = NULL;
	s16_ctor(&name, COUNTERTOP_PREFIX);
	real(status, self, callbacks, id, &name, uid, device);
	s16_dtor(&name);
}
#endif

typedef int (*query_fn)(const void *self, int what, int *value);

int surface_query(const void *self, int what, int *value) __asm__("_ZNK7android7Surface5queryEiPi");
int surface_query(const void *self, int what, int *value) {
	static query_fn real;
	if (!real) real = (query_fn)dlsym(RTLD_NEXT, "_ZNK7android7Surface5queryEiPi");
	int r = real(self, what, value);
	if (r != 0 || what != WINDOW_FORMAT || *value != FORMAT_RGBA_8888) return r;
	int usage = 0;
	if (real(self, WINDOW_CONSUMER_USAGE, &usage) == 0 && (usage & USAGE_VIDEO_ENCODER)) *value = FORMAT_IMPLEMENTATION_DEFINED;
	return r;
}

typedef unsigned (*sleep_fn)(unsigned secs);

static int from_binder(const void *at) {
	Dl_info info;
	if (!dladdr(at, &info) || !info.dli_fname) return 0;
	size_t n = strlen(info.dli_fname), want = sizeof "/libbinder.so" - 1;
	return n >= want && !strcmp(info.dli_fname + n - want, "/libbinder.so");
}

unsigned sleep(unsigned secs) {
	static sleep_fn real;
	if (from_binder(__builtin_return_address(0))) return 0;
	if (!real) real = (sleep_fn)dlsym(RTLD_NEXT, "sleep");
	return real(secs);
}
