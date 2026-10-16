#ifndef PXN_FFB_PROTOCOL_H
#define PXN_FFB_PROTOCOL_H

#include <stdint.h>

#define PXN_FFB_VERSION 1
#define PXN_FFB_COMMAND_SET 1
#define PXN_FFB_COMMAND_STOP 2
#define PXN_FFB_COMMAND_PING 3
#define PXN_FFB_DEFAULT_PORT 21190

#define PXN_FFB_ADAPTER_MASK 0x000F
#define PXN_FFB_ADAPTER_UNKNOWN 0x0000
#define PXN_FFB_ADAPTER_MSFS2020 0x0001
#define PXN_FFB_ADAPTER_MSFS2024 0x0002
#define PXN_FFB_ADAPTER_XPLANE12 0x0003
#define PXN_FFB_ADAPTER_MSFS_GENERIC 0x0004

#define PXN_FFB_STATE_PAUSED 0x0100
#define PXN_FFB_STATE_STOPPED 0x0200
#define PXN_FFB_STATE_EXITING 0x0400

#pragma pack(push, 1)
typedef struct PXNFFBPacket {
    char magic[4];
    uint8_t version;
    uint8_t command;
    uint8_t left;  /* PXN-F19 stick motor; legacy PXNF/1 field name */
    uint8_t right; /* PXN-F19 throttle motor; legacy PXNF/1 field name */
    uint16_t duration_ms;
    uint16_t flags; /* adapter ID in low nibble; simulator state in high byte */
    uint32_t sequence;
} PXNFFBPacket;
#pragma pack(pop)

#if defined(__cplusplus)
static_assert(sizeof(PXNFFBPacket) == 16, "PXNF/1 packet must be 16 bytes");
#endif

#endif
