#ifndef ZIGBOLT_H
#define ZIGBOLT_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

void* zigbolt_transport_create(uint32_t term_length, uint8_t use_hugepages, uint8_t pre_fault);
void zigbolt_transport_destroy(void* handle);

void* zigbolt_ipc_create(const char* name, uint32_t term_length);
void* zigbolt_ipc_open(const char* name, uint32_t term_length);
void zigbolt_ipc_destroy(void* handle);

int32_t zigbolt_publish(void* handle, const uint8_t* data, uint32_t len, int32_t msg_type_id);

typedef void (*zigbolt_fragment_handler_t)(const uint8_t* data, uint32_t len, int32_t msg_type_id);
uint32_t zigbolt_poll(void* handle, zigbolt_fragment_handler_t callback, uint32_t limit);

uint32_t zigbolt_version_major(void);
uint32_t zigbolt_version_minor(void);
uint32_t zigbolt_version_patch(void);

#ifdef __cplusplus
}
#endif

#endif /* ZIGBOLT_H */
