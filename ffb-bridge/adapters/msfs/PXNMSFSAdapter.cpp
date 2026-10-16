#define WIN32_LEAN_AND_MEAN
#include <Windows.h>
#include <SimConnect.h>
#include <WinSock2.h>
#include <WS2tcpip.h>

#include <algorithm>
#include <atomic>
#include <cctype>
#include <cstring>
#include <string>

#include "../../pxn_ffb_protocol.h"

#pragma comment(lib, "SimConnect.lib")
#pragma comment(lib, "Ws2_32.lib")

namespace {
constexpr SIMCONNECT_CLIENT_DATA_ID kClientDataId = 0x2119;
constexpr SIMCONNECT_CLIENT_DATA_DEFINITION_ID kDefinitionId = 0x2119;
constexpr SIMCONNECT_DATA_REQUEST_ID kRequestId = 0x2119;
constexpr SIMCONNECT_CLIENT_EVENT_ID kPauseEventId = 0x2120;
constexpr SIMCONNECT_CLIENT_EVENT_ID kSimStartEventId = 0x2121;
constexpr SIMCONNECT_CLIENT_EVENT_ID kSimStopEventId = 0x2122;
constexpr DWORD kHeartbeatMs = 500;

HANDLE g_simConnect = nullptr;
SOCKET g_socket = INVALID_SOCKET;
sockaddr_in g_service{};
std::atomic_bool g_running{true};
bool g_paused = false;
bool g_simActive = true;
uint16_t g_adapter = PXN_FFB_ADAPTER_MSFS_GENERIC;
DWORD g_lastForwarded = 0;

bool valid_packet(const PXNFFBPacket& packet) {
    return std::memcmp(packet.magic, "PXNF", 4) == 0 &&
           packet.version == PXN_FFB_VERSION;
}

uint16_t detect_adapter(const SIMCONNECT_RECV_OPEN& open) {
    std::string name(open.szApplicationName);
    std::transform(name.begin(), name.end(), name.begin(),
                   [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
    if (name.find("2024") != std::string::npos || open.dwApplicationVersionMajor >= 12) {
        return PXN_FFB_ADAPTER_MSFS2024;
    }
    if (name.find("flight simulator") != std::string::npos ||
        name.find("flightsimulator") != std::string::npos) {
        return PXN_FFB_ADAPTER_MSFS2020;
    }
    return PXN_FFB_ADAPTER_MSFS_GENERIC;
}

void forward_packet(const PXNFFBPacket& packet, uint16_t stateFlags = 0) {
    if (g_socket == INVALID_SOCKET || !valid_packet(packet)) {
        return;
    }
    if (packet.command == PXN_FFB_COMMAND_SET && (g_paused || !g_simActive)) {
        return;
    }
    PXNFFBPacket forwarded = packet;
    constexpr uint16_t knownFlags = PXN_FFB_ADAPTER_MASK | PXN_FFB_STATE_PAUSED |
                                    PXN_FFB_STATE_STOPPED | PXN_FFB_STATE_EXITING;
    forwarded.flags = static_cast<uint16_t>((forwarded.flags & ~knownFlags) |
                                            g_adapter | stateFlags);
    sendto(g_socket, reinterpret_cast<const char*>(&forwarded), sizeof(forwarded), 0,
           reinterpret_cast<const sockaddr*>(&g_service), sizeof(g_service));
    g_lastForwarded = GetTickCount();
}

void send_control(uint8_t command, uint16_t stateFlags = 0) {
    PXNFFBPacket packet{{'P', 'X', 'N', 'F'}, PXN_FFB_VERSION,
                        command, 0, 0, 0, 0, 0};
    forward_packet(packet, stateFlags);
}

void CALLBACK dispatch(SIMCONNECT_RECV* message, DWORD, void*) {
    switch (message->dwID) {
    case SIMCONNECT_RECV_ID_OPEN: {
        const auto* open = reinterpret_cast<const SIMCONNECT_RECV_OPEN*>(message);
        g_adapter = detect_adapter(*open);
        send_control(PXN_FFB_COMMAND_PING);
        break;
    }
    case SIMCONNECT_RECV_ID_CLIENT_DATA: {
        const auto* data = reinterpret_cast<const SIMCONNECT_RECV_CLIENT_DATA*>(message);
        if (data->dwRequestID == kRequestId && data->dwDefineCount > 0) {
            const auto* packet = reinterpret_cast<const PXNFFBPacket*>(&data->dwData);
            forward_packet(*packet);
        }
        break;
    }
    case SIMCONNECT_RECV_ID_EVENT: {
        const auto* event = reinterpret_cast<const SIMCONNECT_RECV_EVENT*>(message);
        switch (event->uEventID) {
        case kPauseEventId:
            g_paused = event->dwData != 0;
            if (g_paused) {
                send_control(PXN_FFB_COMMAND_STOP, PXN_FFB_STATE_PAUSED);
            } else {
                send_control(PXN_FFB_COMMAND_PING);
            }
            break;
        case kSimStartEventId:
            g_simActive = true;
            send_control(PXN_FFB_COMMAND_PING);
            break;
        case kSimStopEventId:
            g_simActive = false;
            send_control(PXN_FFB_COMMAND_STOP, PXN_FFB_STATE_STOPPED);
            break;
        default:
            break;
        }
        break;
    }
    case SIMCONNECT_RECV_ID_QUIT:
        send_control(PXN_FFB_COMMAND_STOP, PXN_FFB_STATE_EXITING);
        g_running = false;
        break;
    default:
        break;
    }
}
} // namespace

int WINAPI wWinMain(HINSTANCE, HINSTANCE, PWSTR, int) {
    WSADATA winsock{};
    if (WSAStartup(MAKEWORD(2, 2), &winsock) != 0) {
        return 2;
    }
    g_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    g_service.sin_family = AF_INET;
    g_service.sin_port = htons(PXN_FFB_DEFAULT_PORT);
    InetPtonW(AF_INET, L"127.0.0.1", &g_service.sin_addr);

    if (FAILED(SimConnect_Open(&g_simConnect, "PXN 2119 FFB Adapter", nullptr, 0, 0, 0))) {
        closesocket(g_socket);
        WSACleanup();
        return 3;
    }

    SimConnect_MapClientDataNameToID(g_simConnect, "PXN2119_FFB", kClientDataId);
    SimConnect_CreateClientData(g_simConnect, kClientDataId, sizeof(PXNFFBPacket),
                                SIMCONNECT_CREATE_CLIENT_DATA_FLAG_DEFAULT);
    SimConnect_AddToClientDataDefinition(g_simConnect, kDefinitionId, 0,
                                         sizeof(PXNFFBPacket), 0.0f,
                                         SIMCONNECT_UNUSED);
    SimConnect_RequestClientData(g_simConnect, kClientDataId, kRequestId,
                                 kDefinitionId, SIMCONNECT_CLIENT_DATA_PERIOD_ON_SET,
                                 SIMCONNECT_CLIENT_DATA_REQUEST_FLAG_CHANGED, 0, 0, 0);
    SimConnect_SubscribeToSystemEvent(g_simConnect, kPauseEventId, "Pause");
    SimConnect_SubscribeToSystemEvent(g_simConnect, kSimStartEventId, "SimStart");
    SimConnect_SubscribeToSystemEvent(g_simConnect, kSimStopEventId, "SimStop");

    while (g_running) {
        SimConnect_CallDispatch(g_simConnect, dispatch, nullptr);
        const DWORD now = GetTickCount();
        if (now - g_lastForwarded >= kHeartbeatMs) {
            uint16_t state = 0;
            if (g_paused) state |= PXN_FFB_STATE_PAUSED;
            if (!g_simActive) state |= PXN_FFB_STATE_STOPPED;
            send_control(PXN_FFB_COMMAND_PING, state);
        }
        Sleep(10);
    }

    send_control(PXN_FFB_COMMAND_STOP, PXN_FFB_STATE_EXITING);
    SimConnect_Close(g_simConnect);
    closesocket(g_socket);
    WSACleanup();
    return 0;
}
