import moment from 'moment';
import { TABLE_DATE_FORMAT } from './constants';

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

export function dateToString(date) {
  return moment(date).format(
    TABLE_DATE_FORMAT);
}

// "field_name" => "Field nNme"
export function fieldToTitle(str) {
  return str.replace(/_/g, ' ').replace(/(?: |\b)(\w)/g, key => key.toUpperCase());

}

export function compareDate(fieldName) {
  return function compare(a, b) {
    const dateA = new Date(a[fieldName]);
    const dateB = new Date(b[fieldName]);
    return dateA - dateB;
  }
}
