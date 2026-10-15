#include "core.h"
#include <assert.h>
#include <limits.h>
#include <math.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

typedef struct {int live;} allocation_state;
static void *tracked_allocate(CFIndex size,CFOptionFlags hint,void *info) {
    (void)hint;
    void *p=malloc((size_t)size);if(p)((allocation_state *)info)->live++;return p;
}
static void tracked_deallocate(void *p,void *info) {if(p){((allocation_state *)info)->live--;free(p);}}
static CFDictionaryRef dictionary(CFAllocatorRef allocator,CFTypeRef value) {
    const void *key=CFSTR("RSSI_CTL_AGR");
    return CFDictionaryCreate(allocator,&key,&value,1,&kCFTypeDictionaryKeyCallBacks,&kCFTypeDictionaryValueCallBacks);
}
int main(void) {
    allocation_state state={0};
    CFAllocatorContext context={0,&state,NULL,NULL,NULL,tracked_allocate,NULL,tracked_deallocate,NULL};
    CFAllocatorRef allocator=CFAllocatorCreate(kCFAllocatorDefault,&context);assert(allocator);
    int64_t parsed=123;
    assert(!beszel_wifi_rssi_owned(NULL,&parsed));
    for(int repeat=0;repeat<1000;repeat++) {
        int64_t values[]={-40,-85,-1,-128,-150,0,1,-151,-999,INT64_MIN,INT64_MAX};
        for(size_t i=0;i<sizeof(values)/sizeof(values[0]);i++) {
            CFNumberRef number=CFNumberCreate(kCFAllocatorDefault,kCFNumberSInt64Type,&values[i]);assert(number);
            CFDictionaryRef property=dictionary(allocator,number);assert(property);CFRelease(number);
            assert(state.live>0);
            bool valid=beszel_wifi_rssi_owned(property,&parsed);
            assert(valid==(i<5));if(valid)assert(parsed==values[i]);assert(state.live==0);
        }
        double floats[]={-40.0,-40.5,NAN,INFINITY,-INFINITY};
        for(size_t i=0;i<sizeof(floats)/sizeof(floats[0]);i++) {
            CFNumberRef floating=CFNumberCreate(NULL,kCFNumberDoubleType,&floats[i]);assert(floating);
            assert(!beszel_wifi_rssi_owned(dictionary(allocator,floating),&parsed));CFRelease(floating);assert(state.live==0);
        }
        int64_t valid=-40;CFNumberRef number=CFNumberCreate(NULL,kCFNumberSInt64Type,&valid);assert(number);
        CFDictionaryRef partial=dictionary(allocator,number);CFRelease(number);
        assert(!beszel_wifi_rssi_owned(partial,NULL));assert(state.live==0);
        assert(!beszel_wifi_rssi_owned(dictionary(allocator,CFSTR("-40")),&parsed));assert(state.live==0);
        assert(!beszel_wifi_rssi_owned(dictionary(allocator,kCFBooleanTrue),&parsed));assert(state.live==0);
        CFDictionaryRef empty=CFDictionaryCreate(allocator,NULL,NULL,0,&kCFTypeDictionaryKeyCallBacks,&kCFTypeDictionaryValueCallBacks);
        assert(!beszel_wifi_rssi_owned(empty,&parsed));assert(state.live==0);
        unsigned char bytes[4096]={0};CFDataRef wrong=CFDataCreate(allocator,bytes,sizeof(bytes));assert(state.live>0);
        assert(!beszel_wifi_rssi_owned(wrong,&parsed));assert(state.live==0);
        wrong=CFDataCreate(allocator,bytes,sizeof(bytes));assert(!beszel_wifi_rssi_owned(wrong,NULL));assert(state.live==0);
    }
    char ssid[33];
    assert(beszel_wifi_ssid(CFSTR("home"),ssid)&&strcmp(ssid,"home")==0);
    assert(beszel_wifi_ssid(CFSTR("网络 café"),ssid)&&strcmp(ssid,"网络 café")==0);
    assert(beszel_wifi_ssid(CFSTR(""),ssid)&&ssid[0]=='\0');
    assert(!beszel_wifi_ssid(NULL,ssid)&&ssid[0]=='\0');
    assert(!beszel_wifi_ssid(kCFBooleanTrue,ssid)&&ssid[0]=='\0');
    assert(!beszel_wifi_ssid(CFSTR("012345678901234567890123456789012"),ssid));
    assert(!beszel_wifi_ssid(CFSTR("网络网络网络网络网络网络网络网络网络网络网络"),ssid));
    UniChar invalid=0xd800;CFStringRef broken=CFStringCreateWithCharacters(NULL,&invalid,1);
    assert(!beszel_wifi_ssid(broken,ssid));CFRelease(broken);
    UniChar nul[]={'a',0,'b'};CFStringRef embedded=CFStringCreateWithCharacters(NULL,nul,3);
    assert(!beszel_wifi_ssid(embedded,ssid));CFRelease(embedded);
    assert(state.live==0);CFRelease(allocator);
    puts("PASS public-CF conversion and owned-container cleanup; no private native API executed");
    return 0;
}
