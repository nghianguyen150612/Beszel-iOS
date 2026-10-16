#include "loader.h"
#include <dlfcn.h>
#include <string.h>

const char *const beszel_wifi_symbols[BESZEL_WIFI_SYMBOL_COUNT] = {
    "WiFiManagerClientCreate", "WiFiManagerClientCopyDevices",
    "WiFiDeviceClientCopyProperty", "WiFiDeviceClientCopyCurrentNetwork",
    "WiFiDeviceClientGetInterfaceName", "WiFiNetworkGetSSID"
};
void beszel_wifi_inventory_read(const char *absolute_path, beszel_wifi_inventory *result) {
    if (!result) return;
    memset(result, 0, sizeof(*result));
    /* dlopen(NULL) opens the main executable: never accept that as a framework. */
    if (!absolute_path || absolute_path[0] != '/') return;
    void *handle = dlopen(absolute_path, RTLD_LAZY | RTLD_LOCAL);
    if (!handle) return;
    result->available = true;
    for (unsigned i = 0; i < BESZEL_WIFI_SYMBOL_COUNT; i++)
        result->symbols[i] = dlsym(handle, beszel_wifi_symbols[i]) != NULL;
    if (dlclose(handle) != 0) memset(result, 0, sizeof(*result));
}
