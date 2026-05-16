#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <dirent.h>

// Arena allocator for automatic memory management
#define CARV_ARENA_BLOCK_SIZE (1024 * 1024)  // 1MB blocks
typedef struct carv_arena_block {
    char* data;
    size_t used;
    size_t capacity;
    struct carv_arena_block* next;
} carv_arena_block;

typedef struct {
    carv_arena_block* head;
    carv_arena_block* current;
} carv_arena;

static carv_arena carv_global_arena = {NULL, NULL};

static carv_arena_block* carv_arena_new_block(size_t min_size) {
    size_t size = min_size > CARV_ARENA_BLOCK_SIZE ? min_size : CARV_ARENA_BLOCK_SIZE;
    carv_arena_block* block = (carv_arena_block*)malloc(sizeof(carv_arena_block));
    block->data = (char*)malloc(size);
    block->used = 0;
    block->capacity = size;
    block->next = NULL;
    return block;
}

static void* carv_arena_alloc(size_t size) {
    size = (size + 7) & ~7;  // 8-byte alignment
    if (!carv_global_arena.current || carv_global_arena.current->used + size > carv_global_arena.current->capacity) {
        carv_arena_block* block = carv_arena_new_block(size);
        if (carv_global_arena.current) {
            carv_global_arena.current->next = block;
        } else {
            carv_global_arena.head = block;
        }
        carv_global_arena.current = block;
    }
    void* ptr = carv_global_arena.current->data + carv_global_arena.current->used;
    carv_global_arena.current->used += size;
    return ptr;
}

static void carv_arena_free_all(void) {
    carv_arena_block* block = carv_global_arena.head;
    while (block) {
        carv_arena_block* next = block->next;
        free(block->data);
        free(block);
        block = next;
    }
    carv_global_arena.head = NULL;
    carv_global_arena.current = NULL;
}

typedef long long carv_int;
typedef double carv_float;
typedef bool carv_bool;
typedef struct { char* data; size_t len; bool owned; } carv_string;

// Create string from C string literal (NOT owned - never freed)
static carv_string carv_string_lit(const char* s) {
    return (carv_string){(char*)s, strlen(s), false};
}

// Create owned string from heap allocation
static carv_string carv_string_own(char* data, size_t len) {
    return (carv_string){data, len, true};
}

// Clone a string (always returns owned copy)
static carv_string carv_string_clone(carv_string s) {
    if (!s.data) return (carv_string){NULL, 0, false};
    char* copy = (char*)carv_arena_alloc(s.len + 1);
    memcpy(copy, s.data, s.len + 1);
    return (carv_string){copy, s.len, true};
}

// Move ownership (source zeroed)
static carv_string carv_string_move(carv_string* s) {
    carv_string out = *s;
    s->data = NULL;
    s->len = 0;
    s->owned = false;
    return out;
}

// Drop a string (free if owned)
static void carv_string_drop(carv_string* s) {
    // Note: with arena allocator, we don't actually free individual strings
    // This is a no-op for now but the interface exists for future per-alloc freeing
    s->data = NULL;
    s->len = 0;
    s->owned = false;
}

static carv_string carv_strdup_str(const char* s) {
    size_t len = strlen(s) + 1;
    char* copy = (char*)carv_arena_alloc(len);
    memcpy(copy, s, len);
    return (carv_string){copy, len - 1, true};
}

typedef struct { carv_int* data; carv_int len; carv_int cap; } carv_int_array;
typedef struct { carv_float* data; carv_int len; carv_int cap; } carv_float_array;
typedef struct { carv_string* data; carv_int len; carv_int cap; } carv_string_array;
typedef struct { carv_bool* data; carv_int len; carv_int cap; } carv_bool_array;

carv_int_array carv_new_int_array(carv_int len) {
    carv_int_array arr;
    arr.data = (carv_int*)carv_arena_alloc(len * sizeof(carv_int));
    arr.len = len;
    arr.cap = len;
    return arr;
}

carv_float_array carv_new_float_array(carv_int len) {
    carv_float_array arr;
    arr.data = (carv_float*)carv_arena_alloc(len * sizeof(carv_float));
    arr.len = len;
    arr.cap = len;
    return arr;
}

carv_string_array carv_new_string_array(carv_int len) {
    carv_string_array arr;
    arr.data = (carv_string*)carv_arena_alloc(len * sizeof(carv_string));
    arr.len = len;
    arr.cap = len;
    return arr;
}

void carv_print_int(carv_int x) { printf("%lld\n", x); }
void carv_print_float(carv_float x) { printf("%g\n", x); }
void carv_print_bool(carv_bool x) { printf("%s\n", x ? "true" : "false"); }
void carv_print_string(carv_string x) { printf("%s\n", x.data); }

void carv_print_int_array(carv_int_array arr) {
    printf("[");
    for (carv_int i = 0; i < arr.len; i++) {
        if (i > 0) printf(", ");
        printf("%lld", arr.data[i]);
    }
    printf("]");
}

void carv_print_float_array(carv_float_array arr) {
    printf("[");
    for (carv_int i = 0; i < arr.len; i++) {
        if (i > 0) printf(", ");
        printf("%g", arr.data[i]);
    }
    printf("]");
}

void carv_print_string_array(carv_string_array arr) {
    printf("[");
    for (carv_int i = 0; i < arr.len; i++) {
        if (i > 0) printf(", ");
        printf("%s", arr.data[i].data);
    }
    printf("]");
}

void carv_print_bool_array(carv_bool_array arr) {
    printf("[");
    for (carv_int i = 0; i < arr.len; i++) {
        if (i > 0) printf(", ");
        printf("%s", arr.data[i] ? "true" : "false");
    }
    printf("]");
}

carv_string carv_read_file(carv_string path) {
    FILE* f = fopen(path.data, "rb");
    if (!f) return (carv_string){NULL, 0, false};
    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    fseek(f, 0, SEEK_SET);
    char* buf = (char*)carv_arena_alloc(len + 1);
    size_t rd = fread(buf, 1, len, f);
    buf[rd] = '\0';
    fclose(f);
    return carv_string_own(buf, rd);
}

carv_bool carv_write_file(carv_string path, carv_string content) {
    FILE* f = fopen(path.data, "wb");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, content.len, f);
    fclose(f);
    return written == content.len;
}

carv_bool carv_file_exists(carv_string path) {
    FILE* f = fopen(path.data, "r");
    if (f) { fclose(f); return true; }
    return false;
}

carv_bool carv_append_file(carv_string path, carv_string content) {
    FILE* f = fopen(path.data, "ab");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, content.len, f);
    fclose(f);
    return written == content.len;
}

carv_bool carv_delete_file(carv_string path) {
    return remove(path.data) == 0;
}

carv_string_array carv_list_dir(carv_string path) {
    carv_string_array arr = {NULL, 0, 0};
    DIR* d = opendir(path.data);
    if (!d) return arr;
    carv_int cap = 16;
    arr.data = (carv_string*)carv_arena_alloc(cap * sizeof(carv_string));
    arr.len = 0;
    arr.cap = cap;
    struct dirent* entry;
    while ((entry = readdir(d)) != NULL) {
        if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) continue;
        if (arr.len >= arr.cap) {
            carv_int new_cap = arr.cap * 2;
            carv_string* new_data = (carv_string*)carv_arena_alloc(new_cap * sizeof(carv_string));
            memcpy(new_data, arr.data, arr.len * sizeof(carv_string));
            arr.data = new_data;
            arr.cap = new_cap;
        }
        arr.data[arr.len++] = carv_strdup_str(entry->d_name);
    }
    closedir(d);
    return arr;
}

carv_int carv_tcp_listen(carv_string host, carv_int port) {
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) return -1;
    int opt = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons((uint16_t)port);
    if (!host.data || host.len == 0 || strcmp(host.data, "0.0.0.0") == 0) {
        addr.sin_addr.s_addr = INADDR_ANY;
    } else if (inet_pton(AF_INET, host.data, &addr.sin_addr) <= 0) {
        close(fd);
        return -1;
    }
    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        close(fd);
        return -1;
    }
    if (listen(fd, 16) < 0) {
        close(fd);
        return -1;
    }
    return fd;
}

carv_int carv_tcp_accept(carv_int listener_fd) {
    int conn_fd = accept((int)listener_fd, NULL, NULL);
    if (conn_fd < 0) return -1;
    return conn_fd;
}

carv_string carv_tcp_read(carv_int conn_fd, carv_int max_bytes) {
    if (max_bytes <= 0) return carv_strdup_str("");
    char* buf = (char*)carv_arena_alloc((size_t)max_bytes + 1);
    ssize_t n = recv((int)conn_fd, buf, (size_t)max_bytes, 0);
    if (n <= 0) {
        buf[0] = '\0';
        return carv_string_own(buf, 0);
    }
    buf[n] = '\0';
    return carv_string_own(buf, (size_t)n);
}

carv_int carv_tcp_write(carv_int conn_fd, carv_string data) {
    if (!data.data || data.len == 0) return 0;
    ssize_t n = send((int)conn_fd, data.data, data.len, 0);
    if (n < 0) return -1;
    return (carv_int)n;
}

carv_bool carv_tcp_close(carv_int fd) {
    return close((int)fd) == 0;
}

carv_string_array carv_split(carv_string str, carv_string sep) {
    carv_string_array arr = {NULL, 0, 0};
    if (!str.data || !sep.data) return arr;
    size_t sep_len = sep.len;
    if (sep_len == 0) {
        arr = carv_new_string_array(1);
        arr.data[0] = carv_string_clone(str);
        return arr;
    }
    // Count occurrences
    carv_int count = 1;
    char* p = str.data;
    while ((p = strstr(p, sep.data)) != NULL) { count++; p += sep_len; }
    arr = carv_new_string_array(count);
    // Split
    char* start = str.data;
    carv_int idx = 0;
    while ((p = strstr(start, sep.data)) != NULL) {
        size_t part_len = p - start;
        char* part = (char*)carv_arena_alloc(part_len + 1);
        memcpy(part, start, part_len);
        part[part_len] = '\0';
        arr.data[idx] = carv_string_own(part, part_len);
        idx++;
        start = p + sep_len;
    }
    size_t tail_len = (size_t)(str.data + str.len - start);
    char* tail = (char*)carv_arena_alloc(tail_len + 1);
    memcpy(tail, start, tail_len);
    tail[tail_len] = '\0';
    arr.data[idx] = carv_string_own(tail, tail_len);
    return arr;
}

carv_string carv_join(carv_string_array arr, carv_string sep) {
    if (arr.len == 0) return carv_strdup_str("");
    size_t sep_len = sep.data ? sep.len : 0;
    size_t total_len = 0;
    for (carv_int i = 0; i < arr.len; i++) {
        if (arr.data[i].data) total_len += arr.data[i].len;
    }
    if (arr.len > 0) total_len += sep_len * (size_t)(arr.len - 1);
    char* result = (char*)carv_arena_alloc(total_len + 1);
    result[0] = '\0';
    for (carv_int i = 0; i < arr.len; i++) {
        if (i > 0 && sep.data) strcat(result, sep.data);
        if (arr.data[i].data) strcat(result, arr.data[i].data);
    }
    return carv_string_own(result, total_len);
}

carv_string carv_trim(carv_string str) {
    if (!str.data) return carv_strdup_str("");
    char* start = str.data;
    char* end = str.data + str.len;
    while (start < end && (*start == ' ' || *start == '\t' || *start == '\n' || *start == '\r')) start++;
    if (start == end) return carv_strdup_str("");
    char* last = end - 1;
    while (last > start && (*last == ' ' || *last == '\t' || *last == '\n' || *last == '\r')) last--;
    size_t len = (size_t)(last - start + 1);
    char* result = (char*)carv_arena_alloc(len + 1);
    memcpy(result, start, len);
    result[len] = '\0';
    return carv_string_own(result, len);
}

carv_string carv_substr(carv_string str, carv_int start, carv_int end) {
    if (!str.data) return carv_strdup_str("");
    size_t str_len = str.len;
    if (start < 0) start = 0;
    if (end < 0) end = (carv_int)str_len;
    if ((size_t)start >= str_len) return carv_strdup_str("");
    if ((size_t)end > str_len) end = (carv_int)str_len;
    if (end <= start) return carv_strdup_str("");
    size_t len = (size_t)(end - start);
    char* result = (char*)carv_arena_alloc(len + 1);
    memcpy(result, str.data + start, len);
    result[len] = '\0';
    return carv_string_own(result, len);
}

carv_string carv_int_to_string(carv_int val) {
    char* buf = (char*)carv_arena_alloc(32);
    int len = snprintf(buf, 32, "%lld", val);
    return carv_string_own(buf, len);
}

carv_string carv_float_to_string(carv_float val) {
    char* buf = (char*)carv_arena_alloc(64);
    int len = snprintf(buf, 64, "%g", val);
    return carv_string_own(buf, len);
}

carv_string carv_bool_to_string(carv_bool val) {
    return carv_strdup_str(val ? "true" : "false");
}

carv_string carv_concat(carv_string a, carv_string b) {
    size_t total = a.len + b.len;
    char* result = (char*)carv_arena_alloc(total + 1);
    memcpy(result, a.data, a.len);
    memcpy(result + a.len, b.data, b.len + 1);
    return carv_string_own(result, total);
}

#ifdef CARV_TARGET_ARM
// ARM implementations provided by vendor HAL at link time
extern void carv_pin_mode(carv_int pin, carv_int mode);
extern void carv_digital_write(carv_int pin, carv_bool value);
extern carv_bool carv_digital_read(carv_int pin);
extern carv_int carv_analog_read(carv_int pin);
extern void carv_analog_write(carv_int pin, carv_int value);
#else
// Host stubs for testing
static void carv_pin_mode(carv_int pin, carv_int mode) { (void)pin; (void)mode; }
static void carv_digital_write(carv_int pin, carv_bool value) { (void)pin; (void)value; }
static carv_bool carv_digital_read(carv_int pin) { (void)pin; return false; }
static carv_int carv_analog_read(carv_int pin) { (void)pin; return 0; }
static void carv_analog_write(carv_int pin, carv_int value) { (void)pin; (void)value; }
#endif

#ifdef CARV_TARGET_ARM
extern carv_int carv_uart_init(carv_int port, carv_int baud);
extern carv_int carv_uart_write(carv_int handle, carv_string data);
extern carv_string carv_uart_read(carv_int handle, carv_int max_bytes);
extern carv_int carv_uart_available(carv_int handle);
#else
static carv_int carv_uart_init(carv_int port, carv_int baud) { (void)port; (void)baud; return 0; }
static carv_int carv_uart_write(carv_int handle, carv_string data) { (void)handle; (void)data; return 0; }
static carv_string carv_uart_read(carv_int handle, carv_int max_bytes) { (void)handle; (void)max_bytes; return carv_string_lit(""); }
static carv_int carv_uart_available(carv_int handle) { (void)handle; return 0; }
#endif

#ifdef CARV_TARGET_ARM
extern carv_int carv_spi_init(carv_int bus, carv_int speed);
extern carv_string carv_spi_transfer(carv_int handle, carv_string data);
extern carv_int carv_spi_write(carv_int handle, carv_string data);
extern carv_string carv_spi_read(carv_int handle, carv_int len);
#else
static carv_int carv_spi_init(carv_int bus, carv_int speed) { (void)bus; (void)speed; return 0; }
static carv_string carv_spi_transfer(carv_int handle, carv_string data) { (void)handle; (void)data; return carv_string_lit(""); }
static carv_int carv_spi_write(carv_int handle, carv_string data) { (void)handle; (void)data; return 0; }
static carv_string carv_spi_read(carv_int handle, carv_int len) { (void)handle; (void)len; return carv_string_lit(""); }
#endif

#ifdef CARV_TARGET_ARM
extern carv_int carv_i2c_init(carv_int bus, carv_int addr);
extern carv_int carv_i2c_write(carv_int handle, carv_string data);
extern carv_string carv_i2c_read(carv_int handle, carv_int len);
#else
static carv_int carv_i2c_init(carv_int bus, carv_int addr) { (void)bus; (void)addr; return 0; }
static carv_int carv_i2c_write(carv_int handle, carv_string data) { (void)handle; (void)data; return 0; }
static carv_string carv_i2c_read(carv_int handle, carv_int len) { (void)handle; (void)len; return carv_string_lit(""); }
#endif

#ifdef CARV_TARGET_ARM
extern carv_int carv_timer_init(carv_int id, carv_int prescaler);
extern void carv_timer_start(carv_int handle);
extern void carv_timer_stop(carv_int handle);
extern carv_int carv_timer_get_count(carv_int handle);
extern void carv_delay_ms(carv_int ms);
extern void carv_delay_us(carv_int us);
#else
static carv_int carv_timer_init(carv_int id, carv_int prescaler) { (void)id; (void)prescaler; return 0; }
static void carv_timer_start(carv_int handle) { (void)handle; }
static void carv_timer_stop(carv_int handle) { (void)handle; }
static carv_int carv_timer_get_count(carv_int handle) { (void)handle; return 0; }
static void carv_delay_ms(carv_int ms) { (void)ms; }
static void carv_delay_us(carv_int us) { (void)us; }
#endif

typedef enum { CARV_TYPE_INT, CARV_TYPE_FLOAT, CARV_TYPE_BOOL, CARV_TYPE_STRING } carv_type_tag;
typedef struct { carv_bool is_ok; carv_type_tag ok_tag; carv_type_tag err_tag; union { carv_int ok_int; carv_float ok_float; carv_bool ok_bool; carv_string ok_str; void* ok_ptr; } ok; union { carv_string err_str; carv_int err_code; } err; } carv_result;

static carv_result carv_ok_int(carv_int val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = true; r.ok_tag = CARV_TYPE_INT; r.ok.ok_int = val; return r; }
static carv_result carv_ok_float(carv_float val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = true; r.ok_tag = CARV_TYPE_FLOAT; r.ok.ok_float = val; return r; }
static carv_result carv_ok_bool(carv_bool val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = true; r.ok_tag = CARV_TYPE_BOOL; r.ok.ok_bool = val; return r; }
static carv_result carv_ok_str(carv_string val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = true; r.ok_tag = CARV_TYPE_STRING; r.ok.ok_str = val; return r; }
static carv_result carv_err_str(carv_string val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = false; r.err_tag = CARV_TYPE_STRING; r.err.err_str = val; return r; }
static carv_result carv_err_code(carv_int val) { carv_result r; memset(&r, 0, sizeof(r)); r.is_ok = false; r.err_tag = CARV_TYPE_INT; r.err.err_code = val; return r; }

// --- Map runtime ---
typedef enum { CARV_MAP_VAL_INT, CARV_MAP_VAL_FLOAT, CARV_MAP_VAL_BOOL, CARV_MAP_VAL_STRING } carv_map_val_tag;
typedef struct { carv_string key; carv_map_val_tag tag; union { carv_int i; carv_float f; carv_bool b; carv_string s; } val; bool occupied; } carv_map_entry;
typedef struct { carv_map_entry* entries; carv_int cap; carv_int len; } carv_map;

static uint64_t carv_map_hash(carv_string key) {
    uint64_t h = 14695981039346656037ULL;
    for (size_t i = 0; i < key.len; i++) {
        h ^= (uint64_t)(unsigned char)key.data[i];
        h *= 1099511628211ULL;
    }
    return h;
}

static carv_map carv_map_new(carv_int cap) {
    carv_map m;
    m.cap = cap < 8 ? 8 : cap;
    m.len = 0;
    m.entries = (carv_map_entry*)carv_arena_alloc(m.cap * sizeof(carv_map_entry));
    memset(m.entries, 0, m.cap * sizeof(carv_map_entry));
    return m;
}

static carv_map_entry* carv_map_find(carv_map* m, carv_string key) {
    uint64_t h = carv_map_hash(key);
    for (carv_int i = 0; i < m->cap; i++) {
        carv_int idx = (carv_int)((h + (uint64_t)i) % (uint64_t)m->cap);
        carv_map_entry* e = &m->entries[idx];
        if (!e->occupied) return e;
        if (e->key.len == key.len && memcmp(e->key.data, key.data, key.len) == 0) return e;
    }
    return NULL;
}

static void carv_map_grow(carv_map* m) {
    carv_int old_cap = m->cap;
    carv_map_entry* old = m->entries;
    m->cap = old_cap * 2;
    m->entries = (carv_map_entry*)carv_arena_alloc(m->cap * sizeof(carv_map_entry));
    memset(m->entries, 0, m->cap * sizeof(carv_map_entry));
    m->len = 0;
    for (carv_int i = 0; i < old_cap; i++) {
        if (old[i].occupied) {
            carv_map_entry* e = carv_map_find(m, old[i].key);
            *e = old[i];
            m->len++;
        }
    }
}

static void carv_map_set_int(carv_map* m, carv_string key, carv_int val) {
    if (m->len * 2 >= m->cap) carv_map_grow(m);
    carv_map_entry* e = carv_map_find(m, key);
    if (!e->occupied) { m->len++; e->occupied = true; e->key = key; }
    e->tag = CARV_MAP_VAL_INT; e->val.i = val;
}

static void carv_map_set_float(carv_map* m, carv_string key, carv_float val) {
    if (m->len * 2 >= m->cap) carv_map_grow(m);
    carv_map_entry* e = carv_map_find(m, key);
    if (!e->occupied) { m->len++; e->occupied = true; e->key = key; }
    e->tag = CARV_MAP_VAL_FLOAT; e->val.f = val;
}

static void carv_map_set_bool(carv_map* m, carv_string key, carv_bool val) {
    if (m->len * 2 >= m->cap) carv_map_grow(m);
    carv_map_entry* e = carv_map_find(m, key);
    if (!e->occupied) { m->len++; e->occupied = true; e->key = key; }
    e->tag = CARV_MAP_VAL_BOOL; e->val.b = val;
}

static void carv_map_set_str(carv_map* m, carv_string key, carv_string val) {
    if (m->len * 2 >= m->cap) carv_map_grow(m);
    carv_map_entry* e = carv_map_find(m, key);
    if (!e->occupied) { m->len++; e->occupied = true; e->key = key; }
    e->tag = CARV_MAP_VAL_STRING; e->val.s = val;
}

static carv_int carv_map_get_int(carv_map* m, carv_string key) {
    carv_map_entry* e = carv_map_find(m, key);
    if (e && e->occupied) return e->val.i;
    return 0;
}

static carv_float carv_map_get_float(carv_map* m, carv_string key) {
    carv_map_entry* e = carv_map_find(m, key);
    if (e && e->occupied) return e->val.f;
    return 0.0;
}

static carv_bool carv_map_get_bool(carv_map* m, carv_string key) {
    carv_map_entry* e = carv_map_find(m, key);
    if (e && e->occupied) return e->val.b;
    return false;
}

static carv_string carv_map_get_str(carv_map* m, carv_string key) {
    carv_map_entry* e = carv_map_find(m, key);
    if (e && e->occupied) return e->val.s;
    return (carv_string){NULL, 0, false};
}

static void carv_print_map(carv_map m) {
    printf("{");
    int first = 1;
    for (carv_int i = 0; i < m.cap; i++) {
        if (!m.entries[i].occupied) continue;
        if (!first) printf(", ");
        first = 0;
        printf("\"%s\": ", m.entries[i].key.data);
        switch (m.entries[i].tag) {
        case CARV_MAP_VAL_INT: printf("%lld", m.entries[i].val.i); break;
        case CARV_MAP_VAL_FLOAT: printf("%g", m.entries[i].val.f); break;
        case CARV_MAP_VAL_BOOL: printf("%s", m.entries[i].val.b ? "true" : "false"); break;
        case CARV_MAP_VAL_STRING: printf("\"%s\"", m.entries[i].val.s.data); break;
        }
    }
    printf("}");
}

carv_int carv_double(carv_int n);
carv_int add(carv_int a, carv_int b);

carv_int carv_double(carv_int n) {
    carv_int __carv_retval = 0;
    __carv_retval = (n * 2);
    goto __carv_exit;
    __carv_exit:;
    return __carv_retval;
}

carv_int add(carv_int a, carv_int b) {
    carv_int __carv_retval = 0;
    __carv_retval = (a + b);
    goto __carv_exit;
    __carv_exit:;
    return __carv_retval;
}


int main(void) {
    (printf("%s", carv_string_lit("Hello, Carv!").data), printf("\n"));
    carv_int x = 10;
    carv_int y = 20;
    (printf("%lld", carv_double(x)), printf("\n"));
    (printf("%lld", add(carv_double(x), 5)), printf("\n"));
    if ((x < y)) {
        (printf("%s", carv_string_lit("x is less than y").data), printf("\n"));
    }
    for (carv_int i = 0; (i < 3); i = (i + 1)) {
        (printf("%s", carv_string_lit("i =").data), printf(" "), printf("%lld", i), printf("\n"));
    }
    carv_int_array nums = (carv_int_array){(carv_int[]){1, 2, 3, 4, 5}, 5, 5};
    for (carv_int __idx_1 = 0; __idx_1 < nums.len; __idx_1++) {
        carv_int n = nums.data[__idx_1];
        (printf("%lld", n), printf("\n"));
    }
    (printf("%s", carv_string_lit("array:").data), printf(" "), carv_print_int_array((carv_int_array){(carv_int[]){1, 2, 3, 4, 5}, 5, 5}), printf("\n"));
    (printf("%s", carv_string_lit("length:").data), printf(" "), printf("%lld", (carv_int_array){(carv_int[]){1, 2, 3, 4, 5}, 5, 5}.len), printf("\n"));
    carv_arena_free_all();
    return 0;
}
