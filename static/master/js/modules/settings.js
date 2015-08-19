
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#channels').length) {
    var timerExists = false;

    var disableAll = function() {
      $("#channels").attr("disabled", true).trigger("chosen:updated");
      $("#groups").attr("disabled", true).trigger("chosen:updated");
      $("#im").attr("disabled", true);
      $("#regexp").attr("disabled", true);
    };
    var enableAll = function() {
      $("#channels").removeAttr("disabled").trigger("chosen:updated");
      $("#groups").removeAttr("disabled").trigger("chosen:updated");
      $("#im").removeAttr("disabled");
      $("#regexp").removeAttr("disabled");
    }


    // Load the channels
    // TODO - add fail handling
    $.getJSON('/info', function(data) {
      var channels = [];
      var groups = [];
      for (var i=0; data.channels && i<data.channels.length; i++) {
        channels.push('<option value="' + data.channels[i].id + '" ' + (data.channels[i].selected ? 'selected' : '') + '>' + data.channels[i].name + '</option>');
      }
      $('#channels').append(channels.join(''));
      for (var i=0; data.groups && i<data.groups.length; i++) {
        groups.push('<option value="' + data.groups[i].id + '" ' + (data.groups[i].selected ? 'selected' : '') + '>' + data.groups[i].name + '</option>');
      }
      $('#groups').append(groups.join(''));
      $('#im').prop('checked', data.im);
      $('#all').prop('checked', data.all);
      $('#regexp').val(data.regexp);
      if (data.regexp) {
        $.ajax({
          type: 'POST',
          url: '/match',
          data: JSON.stringify({regexp: data.regexp}),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(data){
            $('#regexpChannels').html('Will monitor: ' + data.join(', '));
          }
        });
      }
      // If all is enabled then disable all the others
      if (data.all) {
        disableAll();
      }

      // Enable chosen
      $(".chosen-select").chosen({no_results_text: "Oops, No matching entry found:"});

      // Function to save all properties
      var saveAll = function() {
        var save = {};
        save.channels = [];
        save.groups = [];
        save.im = $('#im').is(':checked');
        save.all = $('#all').is(':checked');
        save.regexp = $('#regexp').val();
        $('#channels option:selected').each(function() {
          save.channels.push($(this).val());
        });
        $('#groups option:selected').each(function() {
          save.groups.push($(this).val());
        });

        $.ajax({
          type: 'POST',
          url: '/save',
          data: JSON.stringify(save),
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
            toastr["success"]("Configuration Saved");
          },
          error: function(xhr, status, error) {
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              err += " - " + xhr.responseJSON.errors[0].detail;
            }
            $('#regexpChannels').html(err);
          }
        });
      };
      $('#all').change(function(evt) {
        if ($('#all').is(':checked')) {
          disableAll();
        } else {
          enableAll();
        }
        saveAll();
      });
      $('.chosen-select,#im').change(function(evt) {
        saveAll();
      });

      var saveRegex = function(evt)
      {
        var timer;
        if (evt.type == "keypress") {
          if (!timerExists)
          {
            timerExists = true;
            var timer = setTimeout(function(){
              timerExists = false;
              if (evt && evt.target && evt.target.value) {
                $.ajax({
                  type: 'POST',
                  url: '/match',
                  data: JSON.stringify({regexp: evt.target.value}),
                  headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
                  dataType: 'json',
                  contentType: 'application/json; charset=utf-8',
                  success: function(data){

                    $('#regexpChannels').html('Matched Channels: ' + data.join(', '));
                    saveAll();
                  },
                  error: function(xhr, status, error) {
                    var err = error;
                    if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
                      err += " - " + xhr.responseJSON.errors[0].detail;
                    }
                    $('#regexpChannels').html(err);
                  }
                });
              }
            }, 5000);
          }
        }
        else if (evt.type == "change") {
          if (timerExists) {
            clearTimeout(timer);
            timerExists = false;
          }
          if (evt && evt.target && evt.target.value) {
            $.ajax({
              type: 'POST',
              url: '/match',
              data: JSON.stringify({regexp: evt.target.value}),
              headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
              dataType: 'json',
              contentType: 'application/json; charset=utf-8',
              success: function(data){

                $('#regexpChannels').html('Matched Channels: ' + data.join(', '));
                saveAll();
              },
              error: function(xhr, status, error) {
                var err = error;
                if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
                  err += " - " + xhr.responseJSON.errors[0].detail;
                }
                $('#regexpChannels').html(err);
              }
            });
          }
        }
      }


      $('#regexp').keypress(saveRegex);
      $('#regexp').change(saveRegex);

    });
  }

  // load the feedback widget only on conf page
  $(function (){
    if ($('#channels').length) {
      // get the user information
      $.getJSON('/user', function(data) {
        FreshWidget.init("", {"queryString": "&widgetType=popup&helpdesk_ticket[subject]=Configuration:&helpdesk_ticket[requester]="+data.email, "utf8": "✓",
          "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
          "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );

      });

    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
