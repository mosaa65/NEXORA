export function isRealIOSDevice(device) {
  return Boolean(
    device?.type === "ios" &&
      typeof device.id === "string" &&
      device.id.startsWith("ios_") &&
      !device.id.startsWith("ios_mtp_")
  );
}

export function isWindowsIOSPlaceholder(device) {
  const id = String(device?.id || "");
  const name = String(device?.name || "").toLowerCase();
  if (isRealIOSDevice(device)) {
    return false;
  }
  return (
    id.startsWith("ios_mtp_") ||
    (device?.type === "ios" &&
      (name.includes("iphone") || name.includes("ipad") || name.includes("apple")))
  );
}

export function normalizeTransferDevices(devices = []) {
  const list = Array.isArray(devices) ? devices : [];
  const hasRealIOS = list.some(isRealIOSDevice);

  return list
    .filter((device) => !(hasRealIOS && isWindowsIOSPlaceholder(device)))
    .sort((a, b) => deviceRank(a) - deviceRank(b));
}

function deviceRank(device) {
  if (isRealIOSDevice(device)) return 0;
  if (device?.type === "android") return 1;
  if (device?.type === "storage") return 2;
  if (isWindowsIOSPlaceholder(device)) return 3;
  return 4;
}
