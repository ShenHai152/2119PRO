cmake_minimum_required(VERSION 3.20)
project(PXNMSFSAdapter LANGUAGES CXX)

if(NOT DEFINED MSFS_SDK)
    message(FATAL_ERROR "Set MSFS_SDK to the Microsoft Flight Simulator SDK root")
endif()

add_executable(PXNMSFSAdapter WIN32 PXNMSFSAdapter.cpp)
target_compile_features(PXNMSFSAdapter PRIVATE cxx_std_17)
target_include_directories(PXNMSFSAdapter PRIVATE
    "${MSFS_SDK}/SimConnect SDK/include"
    "${CMAKE_CURRENT_SOURCE_DIR}/../..")
target_link_directories(PXNMSFSAdapter PRIVATE
    "${MSFS_SDK}/SimConnect SDK/lib/static")
target_link_libraries(PXNMSFSAdapter PRIVATE SimConnect Ws2_32)
