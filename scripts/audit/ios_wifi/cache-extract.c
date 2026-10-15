/* Cloud macOS research tool only; never built for or bundled with the Agent.
 * Signature evidence: Apple's dyld-852.2 dsc_extractor.h at revision
 * 3f24a36068a96722cf3acbd5087983ce658e9d70. Reads one cache into a temp directory.
 */
#include <dlfcn.h>
#include <stdio.h>

typedef int (*extract_fn)(const char *, const char *, void (^)(unsigned, unsigned));
int main(int argc, char **argv) {
    if (argc != 3) return 64;
    void *bundle = dlopen("/usr/lib/dsc_extractor.bundle", RTLD_NOW | RTLD_LOCAL);
    if (!bundle) { fputs("Apple dsc_extractor.bundle unavailable\n", stderr); return 77; }
    extract_fn extract = (extract_fn)dlsym(bundle, "dyld_shared_cache_extract_dylibs_progress");
    if (!extract) { dlclose(bundle); return 77; }
    int result = extract(argv[1], argv[2], ^(unsigned current, unsigned total) {});
    dlclose(bundle);
    return result;
}
