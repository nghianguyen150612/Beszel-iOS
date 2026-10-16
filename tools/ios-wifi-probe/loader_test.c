#include "loader.h"
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void absent(const char *path) {
    beszel_wifi_inventory result;
    memset(&result,0xff,sizeof(result));
    beszel_wifi_inventory_read(path,&result);
    assert(!result.available);
    for(unsigned i=0;i<BESZEL_WIFI_SYMBOL_COUNT;i++) assert(!result.symbols[i]);
}
int main(int argc,char **argv) {
    assert(argc==3);
    const char *trace=getenv("BESZEL_WIFI_LOADER_TRACE");assert(trace);
    remove(trace);
    absent(NULL);absent("");absent("relative/library");
    beszel_wifi_inventory_read(argv[1],NULL);
    unsigned expected=(unsigned)atoi(argv[2]);assert(expected<=BESZEL_WIFI_SYMBOL_COUNT);
    for(unsigned repeat=0;repeat<100;repeat++) {
        beszel_wifi_inventory result;
        beszel_wifi_inventory_read(argv[1],&result);
        assert(result.available==(expected!=0));
        unsigned count=0;
        for(unsigned i=0;i<BESZEL_WIFI_SYMBOL_COUNT;i++) count+=result.symbols[i];
        assert(count==expected);
        FILE *file=fopen(trace,"r");
        if(!expected) {assert(!file);continue;}
        assert(file);
        /* A missing dlclose leaves only L here; exit destructors cannot help. */
        for(unsigned j=0;j<=repeat;j++) {assert(fgetc(file)=='L');assert(fgetc(file)=='U');}
        assert(fgetc(file)==EOF);assert(fclose(file)==0);
    }
    puts("PASS production loader: 100 inventories, immediate unload, no private call");
    return 0;
}
