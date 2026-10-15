/* Test dylib only. Deliberately unsuitable prototypes, abort if ever invoked.
 * The inventory must never turn symbol-name presence into an API invocation.
 */
#include <stdlib.h>
#include <stdio.h>
/* Test-only trace. The test checks unload before returning from inventory,
 * so process-exit cleanup cannot conceal a missing production dlclose. */
static void trace(char event) {
    const char *path=getenv("BESZEL_WIFI_LOADER_TRACE");
    if (!path) return;
    FILE *file=fopen(path,"a");
    if (!file) abort();
    if (fputc(event,file)==EOF || fclose(file)!=0) abort();
}
__attribute__((constructor)) static void loaded(void) {trace('L');}
__attribute__((destructor)) static void unloaded(void) {trace('U');}
void WiFiManagerClientCreate(void) {abort();}
#ifndef BESZEL_PARTIAL_SYMBOLS
void WiFiManagerClientCopyDevices(void) {abort();}
void WiFiDeviceClientCopyProperty(void) {abort();}
void WiFiDeviceClientCopyCurrentNetwork(void) {abort();}
void WiFiDeviceClientGetInterfaceName(void) {abort();}
void WiFiNetworkGetSSID(void) {abort();}
#endif
