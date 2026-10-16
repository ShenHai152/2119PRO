#define WIN32_LEAN_AND_MEAN
#include <WinSock2.h>
#include <WS2tcpip.h>

#include <algorithm>
#include <cstdint>
#include <cstring>

#include "XPLMDataAccess.h"
#include "XPLMPlugin.h"
#include "XPLMProcessing.h"
#include "../../pxn_ffb_protocol.h"

namespace {
float g_stick = 0.0f;
float g_throttle = 0.0f;
int g_enabled = 0;
int g_durationMs = 100;
uint32_t g_sequence = 1;
bool g_wasEnabled = false;
bool g_wasPaused = false;
float g_lastPacketTime = 0.0f;
SOCKET g_socket = INVALID_SOCKET;
sockaddr_in g_service{};
XPLMDataRef g_refs[6]{};
XPLMDataRef g_pausedRef = nullptr;

float read_float(void* ref) { return *static_cast<float*>(ref); }
void write_float(void* ref, float value) { *static_cast<float*>(ref) = value; }
int read_int(void* ref) { return *static_cast<int*>(ref); }
void write_int(void* ref, int value) { *static_cast<int*>(ref) = value; }

void send_packet(uint8_t command, uint8_t stick, uint8_t throttle,
                 uint16_t duration, uint16_t stateFlags = 0) {
    if (g_socket == INVALID_SOCKET) {
        return;
    }
    PXNFFBPacket packet{{'P', 'X', 'N', 'F'}, PXN_FFB_VERSION, command,
                        stick, throttle, duration,
                        static_cast<uint16_t>(PXN_FFB_ADAPTER_XPLANE12 | stateFlags),
                        g_sequence++};
    sendto(g_socket, reinterpret_cast<const char*>(&packet), sizeof(packet), 0,
           reinterpret_cast<const sockaddr*>(&g_service), sizeof(g_service));
    g_lastPacketTime = XPLMGetElapsedTime();
}

float flight_loop(float, float, int, void*) {
    const float now = XPLMGetElapsedTime();
    const bool paused = g_pausedRef != nullptr && XPLMGetDatai(g_pausedRef) != 0;
    if (paused) {
        if (!g_wasPaused || g_wasEnabled) {
            send_packet(PXN_FFB_COMMAND_STOP, 0, 0, 0, PXN_FFB_STATE_PAUSED);
        } else if (now - g_lastPacketTime >= 0.5f) {
            send_packet(PXN_FFB_COMMAND_PING, 0, 0, 0, PXN_FFB_STATE_PAUSED);
        }
        g_wasPaused = true;
        g_wasEnabled = false;
        return 0.02f;
    }
    if (g_wasPaused) {
        send_packet(PXN_FFB_COMMAND_PING, 0, 0, 0);
        g_wasPaused = false;
    }
    if (g_enabled != 0) {
        const auto stick = static_cast<uint8_t>(std::clamp(g_stick, 0.0f, 1.0f) * 255.0f);
        const auto throttle = static_cast<uint8_t>(std::clamp(g_throttle, 0.0f, 1.0f) * 255.0f);
        const auto duration = static_cast<uint16_t>(std::clamp(g_durationMs, 1, 5000));
        send_packet(PXN_FFB_COMMAND_SET, stick, throttle, duration);
        g_wasEnabled = true;
    } else if (g_wasEnabled) {
        send_packet(PXN_FFB_COMMAND_STOP, 0, 0, 0);
        g_wasEnabled = false;
    } else if (now - g_lastPacketTime >= 0.5f) {
        send_packet(PXN_FFB_COMMAND_PING, 0, 0, 0);
    }
    return 0.02f;
}
} // namespace

PLUGIN_API int XPluginStart(char* name, char* signature, char* description) {
    std::strcpy(name, "PXN FFB Interface");
    std::strcpy(signature, "pxn.2119.ffb.interface");
    std::strcpy(description, "Forwards simulator-owned vibration commands to PXN service");

    WSADATA winsock{};
    if (WSAStartup(MAKEWORD(2, 2), &winsock) != 0) {
        return 0;
    }
    g_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    g_service.sin_family = AF_INET;
    g_service.sin_port = htons(PXN_FFB_DEFAULT_PORT);
    InetPtonW(AF_INET, L"127.0.0.1", &g_service.sin_addr);
    g_pausedRef = XPLMFindDataRef("sim/time/paused");

    g_refs[0] = XPLMRegisterDataAccessor("pxn/ffb/stick", xplmType_Float, 1,
        nullptr, nullptr, read_float, write_float, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_stick, &g_stick);
    g_refs[1] = XPLMRegisterDataAccessor("pxn/ffb/throttle", xplmType_Float, 1,
        nullptr, nullptr, read_float, write_float, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_throttle, &g_throttle);
    g_refs[2] = XPLMRegisterDataAccessor("pxn/ffb/left", xplmType_Float, 1,
        nullptr, nullptr, read_float, write_float, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_stick, &g_stick);
    g_refs[3] = XPLMRegisterDataAccessor("pxn/ffb/right", xplmType_Float, 1,
        nullptr, nullptr, read_float, write_float, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_throttle, &g_throttle);
    g_refs[4] = XPLMRegisterDataAccessor("pxn/ffb/enabled", xplmType_Int, 1,
        read_int, write_int, nullptr, nullptr, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_enabled, &g_enabled);
    g_refs[5] = XPLMRegisterDataAccessor("pxn/ffb/duration_ms", xplmType_Int, 1,
        read_int, write_int, nullptr, nullptr, nullptr, nullptr,
        nullptr, nullptr, nullptr, nullptr, nullptr, nullptr, &g_durationMs, &g_durationMs);
    XPLMRegisterFlightLoopCallback(flight_loop, 0.02f, nullptr);
    send_packet(PXN_FFB_COMMAND_PING, 0, 0, 0);
    return 1;
}

PLUGIN_API void XPluginStop() {
    send_packet(PXN_FFB_COMMAND_STOP, 0, 0, 0, PXN_FFB_STATE_EXITING);
    XPLMUnregisterFlightLoopCallback(flight_loop, nullptr);
    for (auto ref : g_refs) {
        if (ref) XPLMUnregisterDataAccessor(ref);
    }
    if (g_socket != INVALID_SOCKET) closesocket(g_socket);
    WSACleanup();
}

PLUGIN_API int XPluginEnable() { return 1; }
PLUGIN_API void XPluginDisable() {
    g_enabled = 0;
    send_packet(PXN_FFB_COMMAND_STOP, 0, 0, 0, PXN_FFB_STATE_STOPPED);
}
PLUGIN_API void XPluginReceiveMessage(XPLMPluginID, int, void*) {}
