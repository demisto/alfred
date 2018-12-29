const MD5Mask = 1;
const URLMask = 2;
const IPMask = 4;
const FILEMask = 8;

export function parseType(type) {
  return {
    isMD5: !!(type && MD5Mask),
    isURL: !!(type && URLMask),
    isIP: !!(type && IPMask),
    isFile: !!(type && FILEMask),
  };
}

export function keysToString(obj, defaultVal = 'Unknown') {
  if (!obj) {
    return defaultVal;
  }

  return Object.keys(obj).join(', ') || defaultVal;
}

