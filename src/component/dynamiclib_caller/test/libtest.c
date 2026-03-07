#include <stdio.h>
#include <string.h>

int offset = 0;

// 简单的加法函数
__attribute__((visibility("default"))) int add(int a, int b) {
    return a + b + offset;
}

// 简单的减法函数
__attribute__((visibility("default"))) int sub(int a, int b) {
    return a - b + offset;
}

// 函数注入
__attribute__((visibility("default"))) void injector_func(int (*func)()) {
    offset = func();
}