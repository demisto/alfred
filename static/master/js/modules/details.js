
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on details page
  if ($('#details').length) {
    var uri = new URI();
    var qParts = uri.search(true);
    if (qParts['u']) {
      if (qParts['f']) {
        $('#for').text('file ' + qParts['f']);
      } else if (qParts['m']) {

      }
    }
  }
})(window.jQuery);

// END Settings Handler
// -----------------------------------
