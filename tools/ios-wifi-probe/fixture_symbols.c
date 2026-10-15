/* Test dylib only. Deliberately unsuitable prototypes, abort if ever invoked.
 * The inventory must never turn symbol-name presence into an API invocation.
 */
#include <stdlib.h>
void WiFiManagerClientCreate(void) {abort();}
#ifndef BESZEL_PARTIAL_SYMBOLS
void WiFiManagerClientCopyDevices(void) {abort();}
void WiFiDeviceClientCopyProperty(void) {abort();}
void WiFiDeviceClientCopyCurrentNetwork(void) {abort();}
void WiFiDeviceClientGetInterfaceName(void) {abort();}
void WiFiNetworkGetSSID(void) {abort();}
#endif
