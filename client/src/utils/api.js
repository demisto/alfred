import { API_RESPONSE_STATUS } from './constants';

export async function get(url) {
  try {
    const response = await fetch(url, {
        headers: { 'Accept': 'application/json' }
    });

    const data = await response.json();

    // convert response status to success/fail
    const status = (response.status >= 200 && response.status < 400) ? API_RESPONSE_STATUS.success
      : API_RESPONSE_STATUS.error;

    return {
      status,
      data
    }
  } catch (err) {
      return {
        status: API_RESPONSE_STATUS.error,
        data: err
      }
  }
}
