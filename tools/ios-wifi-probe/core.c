#include "core.h"
#include <string.h>

bool beszel_wifi_rssi_borrowed(CFTypeRef property, int64_t *value) {
    if (!property || !value || CFGetTypeID(property) != CFDictionaryGetTypeID()) return false;
    CFTypeRef number = CFDictionaryGetValue((CFDictionaryRef)property, CFSTR("RSSI_CTL_AGR"));
    if (!number || CFGetTypeID(number) != CFNumberGetTypeID() || CFNumberIsFloatType((CFNumberRef)number)) return false;
    int64_t raw = 0;
    if (!CFNumberGetValue((CFNumberRef)number, kCFNumberSInt64Type, &raw) || raw < -150 || raw > -1) return false;
    *value = raw;
    return true;
}

/* Consume one established owned CF reference, including on partial failure.
 * Its borrowed child stays alive until numeric extraction is finished.
 * This does NOT prove ownership of private manager/device/network objects.
 */
bool beszel_wifi_rssi_owned(CFTypeRef property, int64_t *value) {
    bool valid = beszel_wifi_rssi_borrowed(property, value);
    if (property) CFRelease(property);
    return valid;
}

bool beszel_wifi_ssid(CFTypeRef string, char output[33]) {
    if (!output) return false;
    output[0] = '\0';
    if (!string || CFGetTypeID(string) != CFStringGetTypeID()) return false;
    CFIndex length = CFStringGetLength((CFStringRef)string);
    if (length < 0 || length > 32) return false;
    for (CFIndex i=0;i<length;i++) if (CFStringGetCharacterAtIndex((CFStringRef)string,i)==0) return false;
    if (!CFStringGetCString((CFStringRef)string, output, 33, kCFStringEncodingUTF8)) {
        output[0] = '\0';
        return false;
    }
    return true;
}
