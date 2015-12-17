
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // load the feedback widget only on conf page
  $(function () {
    if ($('#configdiv').length) {
      // get the user information
      $.getJSON('/user', function(data) {
        $('#realname').text(data.real_name);
        $('#useremail').text(data.email);
        $('#teamname').text('Team: ' + data.team_name);
        zE(function() {
          zE.identify({
            name: data.real_name,
            email: data.email
          });
        });
      })
      .error( function(xhr, status, error) {
        var err = error;
        if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
          err += " - " + xhr.responseJSON.errors[0].detail;
          if (xhr.responseJSON.errors[0].status == 401) {
            // unauthorized user - user not logged in
            $('#unauthmodal').modal('show');
            window.setTimeout(function() {
              $('#unauthmodal').modal('hide');
              window.location.replace("/");
            }, 3000);
          }
        }
      });
    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
