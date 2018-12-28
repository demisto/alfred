export const API_RESPONSE_STATUS = {
  success: 'success',
  error: 'err',
};


export async function get(url) {
  try {
    const response = await fetch(url, {
        headers: { 'Accept': 'application/json' }
    });

    const data = await response.json();
    return {
      status: API_RESPONSE_STATUS.success,
      data
    }
  } catch (err) {
      return {
        status: API_RESPONSE_STATUS.error,
        data: err
      }
  }
}
