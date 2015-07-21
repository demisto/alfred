
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';
  // Run this only on conf
  if ($('#channels').length) {
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
      $('#regexp').val(data.regexp);
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

      // Enable chosen
      $(".chosen-select").chosen({no_results_text: "Oops, No matching entry found:"});
      var saveAll = function() {
        var save = {};
        save.channels = [];
        save.groups = [];
        save.im = $('#im').is(':checked');
        save.regexp = $('#regexp').val();
        $('#channels option:selected').each(function() {
          save.channels.push($(this).val());
        });
        $('#groups option:selected').each(function() {
          save.groups.push($(this).val());
        });
        // TODO - handle error
        $.ajax({
          type: 'POST',
          url: '/save',
          data: JSON.stringify(save),
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(){
            // TODO - clear the toaster
          }
        });
      };
      $('.chosen-select,#im').change(function(evt) {
        saveAll();
      });
      $('#regexp').change(function(evt) {
        // First, validate and load the affected channels
        if (evt && evt.target && evt.target.value) {
          $.ajax({
            type: 'POST',
            url: '/match',
            data: JSON.stringify({regexp: evt.target.value}),
            headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
            dataType: 'json',
            contentType: 'application/json; charset=utf-8',
            success: function(data){
              $('#regexpChannels').html('Will monitor: ' + data.join(', '));
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
      });
    });
  }

  // load the feedback widget only on conf page
  $(function (){
    if ($('#channels').length) {
      FreshWidget.init("", {"queryString": "&widgetType=popup", "utf8": "✓",
        "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
        "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );
    }
  });
})(window.jQuery);

// END Settings Handler
// -----------------------------------
