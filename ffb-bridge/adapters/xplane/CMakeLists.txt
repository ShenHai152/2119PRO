cmake_minimum_required(VERSION 3.20)
project(PXNXPlaneAdapter LANGUAGES CXX)

if(NOT DEFINED XPLANE_SDK)
    message(FATAL_ERROR "Set XPLANE_SDK to the X-Plane SDK root")
endif()

add_library(PXNXPlaneAdapter MODULE PXNXPlaneAdapter.cpp)
target_compile_features(PXNXPlaneAdapter PRIVATE cxx_std_17)
target_compile_definitions(PXNXPlaneAdapter PRIVATE IBM=1 XPLM200=1 XPLM210=1 XPLM300=1 XPLM301=1)
target_include_directories(PXNXPlaneAdapter PRIVATE
    "${XPLANE_SDK}/CHeaders/XPLM"
    "${XPLANE_SDK}/CHeaders/Widgets"
    "${CMAKE_CURRENT_SOURCE_DIR}/../..")
target_link_directories(PXNXPlaneAdapter PRIVATE "${XPLANE_SDK}/Libraries/Win")
target_link_libraries(PXNXPlaneAdapter PRIVATE XPLM_64 Ws2_32)
set_target_properties(PXNXPlaneAdapter PROPERTIES PREFIX "" SUFFIX ".xpl")
