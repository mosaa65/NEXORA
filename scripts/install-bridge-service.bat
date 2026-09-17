@echo off
chcp 65001 >nul
echo ========================================================
echo   NEXORA USB Copy Bridge - تثبيت خدمة الويندوز التلقائية
echo ========================================================
echo.

:: 1. التحقق من صلاحيات المسؤول (Administrator)
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] يجب تشغيل هذا الملف كمسؤول (Run as Administrator)!
    echo اضغط بزر الفأرة الأيمن على الملف واختر "تشغيل كمسؤول".
    echo.
    pause
    exit /b 1
)

:: 2. تحديد مسار الملف التنفيذي للخدمة
set "SCRIPT_DIR=%~dp0"
set "BIN_PATH=%SCRIPT_DIR%..\server\nexora-bridge.exe"

if not exist "%BIN_PATH%" (
    set "BIN_PATH=%SCRIPT_DIR%nexora-bridge.exe"
)

if not exist "%BIN_PATH%" (
    echo [ERROR] تعذر العثور على الملف التنفيذي nexora-bridge.exe!
    echo تأكد من بناء البرنامج أو وضع nexora-bridge.exe بجانب هذا السكريبت.
    echo.
    pause
    exit /b 1
)

:: تحويل المسار إلى مسار كامل
for %%i in ("%BIN_PATH%") do set "FULL_BIN_PATH=%%~fi"

echo [1/4] إيقاف وحذف أي نسخة قديمة من الخدمة...
sc.exe stop NEXORACopyBridge >nul 2>&1
sc.exe delete NEXORACopyBridge >nul 2>&1
timeout /t 2 /nobreak >nul

echo [2/4] تسجيل الخدمة في نظام ويندوز عبر sc.exe...
sc.exe create NEXORACopyBridge binPath= "\"%FULL_BIN_PATH%\" -service" start= auto DisplayName= "NEXORA USB Copy Bridge"
if %errorLevel% neq 0 (
    echo [ERROR] فشل تسجيل الخدمة عبر sc.exe!
    pause
    exit /b %errorLevel%
)

sc.exe description NEXORACopyBridge "خدمة NEXORA المحلية للنسخ إلى أجهزة USB والهواتف الذكية"

:: 3. ضبط إعادة التشغيل التلقائي عند أي تعثر
sc.exe failure NEXORACopyBridge reset= 86400 actions= restart/5000/restart/10000/restart/15000 >nul 2>&1

echo [3/4] السماح بالمنفذ المحلي في جدار الحماية (Firewall)...
netsh advfirewall firewall add rule name="NEXORA Copy Bridge" dir=in action=allow protocol=TCP localport=32145 >nul 2>&1

echo [4/4] بدء تشغيل الخدمة...
sc.exe start NEXORACopyBridge
if %errorLevel% neq 0 (
    echo [WARNING] لم تتمكن الخدمة من البدء فوراً. تحقق من سجلات ويندوز.
) else (
    echo.
    echo ========================================================
    echo   تم تثبيت وبدء خدمة NEXORA Copy Bridge بنجاح!
    echo   الخدمة ستعمل الآن تلقائياً مع كل إقلاع للجهاز بالخلفية.
    echo ========================================================
)

echo.
pause
