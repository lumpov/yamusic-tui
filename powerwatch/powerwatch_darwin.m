//go:build darwin

#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <CoreGraphics/CoreGraphics.h>
#include <pthread.h>

extern void goPowerDisplayOff(void);
extern void goPowerDisplayOn(void);
extern void goPowerSessionLocked(void);
extern void goPowerSessionUnlocked(void);

@interface YTPowerWatch : NSObject
@end

@implementation YTPowerWatch
- (void)screensDidSleep:(NSNotification *)n { goPowerDisplayOff(); }
- (void)screensDidWake:(NSNotification *)n { goPowerDisplayOn(); }
- (void)screenLocked:(NSNotification *)n { goPowerSessionLocked(); }
- (void)screenUnlocked:(NSNotification *)n { goPowerSessionUnlocked(); }
@end

static pthread_mutex_t powerMu = PTHREAD_MUTEX_INITIALIZER;
static pthread_t powerThread;
static volatile int powerRunning;
static YTPowerWatch *powerWatcher;

int queryScreenLocked(void) {
    CFDictionaryRef dict = CGSessionCopyCurrentDictionary();
    if (!dict) {
        return 0;
    }
    const void *v = CFDictionaryGetValue(dict, CFSTR("CGSSessionScreenIsLocked"));
    int locked = v && CFBooleanGetValue((CFBooleanRef)v);
    CFRelease(dict);
    return locked;
}

static void *powerThreadMain(void *arg) {
    @autoreleasepool {
        powerWatcher = [YTPowerWatch new];
        NSNotificationCenter *ws = [[NSWorkspace sharedWorkspace] notificationCenter];
        [ws addObserver:powerWatcher selector:@selector(screensDidSleep:) name:NSWorkspaceScreensDidSleepNotification object:nil];
        [ws addObserver:powerWatcher selector:@selector(screensDidWake:) name:NSWorkspaceScreensDidWakeNotification object:nil];
        NSDistributedNotificationCenter *dist = [NSDistributedNotificationCenter defaultCenter];
        [dist addObserver:powerWatcher selector:@selector(screenLocked:) name:@"com.apple.screenIsLocked" object:nil];
        [dist addObserver:powerWatcher selector:@selector(screenUnlocked:) name:@"com.apple.screenIsUnlocked" object:nil];
        while (powerRunning) {
            [[NSRunLoop currentRunLoop] runMode:NSDefaultRunLoopMode beforeDate:[NSDate dateWithTimeIntervalSinceNow:0.5]];
        }
        [ws removeObserver:powerWatcher];
        [dist removeObserver:powerWatcher];
        powerWatcher = nil;
    }
    return NULL;
}

void startPowerWatch(void) {
    pthread_mutex_lock(&powerMu);
    if (!powerRunning) {
        powerRunning = 1;
        pthread_create(&powerThread, NULL, powerThreadMain, NULL);
    }
    pthread_mutex_unlock(&powerMu);
}

void stopPowerWatch(void) {
    pthread_t t = 0;
    pthread_mutex_lock(&powerMu);
    if (powerRunning) {
        powerRunning = 0;
        t = powerThread;
        powerThread = 0;
    }
    pthread_mutex_unlock(&powerMu);
    if (t) {
        pthread_join(t, NULL);
    }
}
