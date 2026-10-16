#ifndef BESZEL_WIFI_LOADER_H
#define BESZEL_WIFI_LOADER_H
#include <stdbool.h>
#define BESZEL_WIFI_SYMBOL_COUNT 6
extern const char *const beszel_wifi_symbols[BESZEL_WIFI_SYMBOL_COUNT];
typedef struct {
    bool available;
    bool symbols[BESZEL_WIFI_SYMBOL_COUNT];
} beszel_wifi_inventory;
/* Inventory only: no private function-pointer cast or invocation. */
void beszel_wifi_inventory_read(const char *absolute_path, beszel_wifi_inventory *result);
#endif
