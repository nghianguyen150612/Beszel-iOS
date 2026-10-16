#ifndef BESZEL_WIFI_CORE_H
#define BESZEL_WIFI_CORE_H
#include <CoreFoundation/CoreFoundation.h>
#include <stdbool.h>
#include <stdint.h>
#include <stddef.h>

/* Public CF objects only. No private opaque object dereference or API call. */
bool beszel_wifi_rssi_borrowed(CFTypeRef property, int64_t *value);
bool beszel_wifi_rssi_owned(CFTypeRef property, int64_t *value);
bool beszel_wifi_ssid(CFTypeRef string, char output[33]);
#endif
