@echo off
chcp 65001 >nul
echo ========================================================
echo   NEXORA USB Copy Bridge - إلغاء تثبيت خدمة الويندوز
echo ========================================================
echo.

:: 1. التحقق من صلاحيات المسؤول
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] يجب تشغيل هذا الملف كمسؤول (Run as Administrator)!
    echo.
    pause
    exit /b 1
)

echo [1/3] إيقاف خدمة NEXORACopyBridge...
sc.exe stop NEXORACopyBridge >nul 2>&1
timeout /t 2 /nobreak >nul

echo [2/3] حذف الخدمة من نظام ويندوز...
sc.exe delete NEXORACopyBridge
if %errorLevel% neq 0 (
    echo [WARNING] لم يتم العثور على الخدمة أو تعذر حذفها.
) else (
    echo تم حذف الخدمة بنجاح.
)

echo [3/3] إزالة قاعدة جدار الحماية...
netsh advfirewall firewall delete rule name="NEXORA Copy Bridge" >nul 2>&1

echo.
echo ========================================================
echo   تم إلغاء تثبيت خدمة NEXORA Copy Bridge بالكامل.
echo ========================================================
echo.
pause
