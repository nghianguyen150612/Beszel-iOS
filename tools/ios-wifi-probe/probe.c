/* Separate disabled diagnostic. Never linked or bundled into Agent/Hub.
 * There is intentionally no private function-pointer cast or invocation here.
 */
#include "core.h"
#include <dlfcn.h>
#include <stdio.h>
#include <string.h>
#include <sys/sysctl.h>

#ifndef BESZEL_WIFI_REVISION
#define BESZEL_WIFI_REVISION "unknown"
#endif
static const char build_info[] = "beszel-ios-wifi-probe revision=" BESZEL_WIFI_REVISION " protocol=1 activation=disabled";
#ifndef BESZEL_WIFI_FRAMEWORK_PATH
#define BESZEL_WIFI_FRAMEWORK_PATH "/System/Library/PrivateFrameworks/MobileWiFi.framework/MobileWiFi"
#endif
static const char framework[] = BESZEL_WIFI_FRAMEWORK_PATH;
static const char *symbols[] = {"WiFiManagerClientCreate", "WiFiManagerClientCopyDevices", "WiFiDeviceClientCopyProperty", "WiFiDeviceClientCopyCurrentNetwork", "WiFiDeviceClientGetInterfaceName", "WiFiNetworkGetSSID"};

static void json_string(const char *text) {
    putchar('"');
    for (const unsigned char *p=(const unsigned char *)text; *p; p++) {
        if (*p=='"' || *p=='\\') printf("\\%c",*p);
        else if (*p<32 || *p>126) printf("\\u%04x",*p);
        else putchar(*p);
    }
    putchar('"');
}
static void identity_value(const char *key) {
    char value[64] = {0}; size_t size=sizeof(value);
    if (sysctlbyname(key,value,&size,NULL,0)!=0 || size>sizeof(value)) value[0]='\0';
    value[sizeof(value)-1]='\0'; json_string(value);
}
static void identity(void) {
    printf("\"model\":");identity_value("hw.machine");
    printf(",\"os_build\":");identity_value("kern.osversion");
    printf(",\"revision\":");json_string(BESZEL_WIFI_REVISION);
}
int main(int argc,char **argv) {
    if (argc!=2) return 64;
    if (strcmp(argv[1],"--snapshot-v1")==0) {
        puts("{\"version\":1,\"source\":\"mobilewifi-ios12\",\"status\":\"unavailable\",\"reason\":\"native-contract-unverified\",\"devices\":[]}");
        return 0;
    }
    if (strcmp(argv[1],"--build-info-v1")==0) {puts(build_info);return 0;}
    if (strcmp(argv[1],"--identity-v1")==0) {printf("{\"version\":1,");identity();puts("}");return 0;}
    if (strcmp(argv[1],"--symbols-v1")==0) {
        void *handle=dlopen(framework,RTLD_LAZY|RTLD_LOCAL);
        printf("{\"version\":1,");identity();printf(",\"framework_available\":%s,\"symbols\":{",handle?"true":"false");
        for (size_t i=0;i<sizeof(symbols)/sizeof(symbols[0]);i++) {
            if(i)putchar(',');json_string(symbols[i]);printf(":%s",handle&&dlsym(handle,symbols[i])?"true":"false");
        }
        puts("},\"private_calls_executed\":false,\"acquisition_enabled\":false}");
        if(handle)dlclose(handle);
        return 0;
    }
    return 64;
}
