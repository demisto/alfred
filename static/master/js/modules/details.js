
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

        $.ajax({
          type: 'GET',
          url: '/work',
          data: qParts,
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(){
            toastr.options = {
              "closeButton": false,
              "debug": false,
              "newestOnTop": false,
              "progressBar": false,
              "positionClass": "toast-top-center",
              "preventDuplicates": true,
              "onclick": null,
              "showDuration": "300",
              "hideDuration": "1000",
              "timeOut": "5000",
              "extendedTimeOut": "1000",
              "showEasing": "swing",
              "hideEasing": "linear",
              "showMethod": "fadeIn",
              "hideMethod": "fadeOut"
            };
            toastr["success"]("ajax request succeded");
          }
        });

      }
    }
  }
})(window.jQuery);

// END Settings Handler
// -----------------------------------
