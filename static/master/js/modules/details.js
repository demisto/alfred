
// Settings Handler
// -----------------------------------

(function ($) {
  'use strict';

  // Run this only on details page
  if ($('#details').length) {
    var MD5Mask = 1;
    var URLMask = 2;
    var IPMask = 4;
    var FILEMask = 8;

    var isfile;
    var ismd5;
    var isurl;
    var isip;

    var uri = new URI();
    var qParts = uri.search(true);

    // freshdesk widget
    $.ajax({
      type: 'GET',
      url: '/user',
      data: "",
      dataType: 'json',
      contentType: 'application/json; charset=utf-8',
      success: function(data) {
        FreshWidget.init("", {"queryString": "&widgetType=popup&searchArea=no&helpdesk_ticket[subject]=Details:&helpdesk_ticket[requester]="+data.email, "utf8": "✓",
          "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
          "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );
      },
      error: function(xhr, status, error) {
        FreshWidget.init("", {"queryString": "&widgetType=popup&searchArea=no&helpdesk_ticket[subject]=Details:", "utf8": "✓",
          "widgetType": "popup", "buttonType": "text", "buttonText": "Feedback", "buttonColor": "white", "buttonBg": "#006063",
          "alignment": "2", "offset": "500px", "formHeight": "500px", "url": "https://demisto.freshdesk.com"} );
      }
    });



    // Details for the ip address query
    var IpDiv = React.createClass({
      resultmessage: function() {
        var resultMessage;
        var ipdata = this.props.data.ip;
        if (ipdata.Result == 0)
        {
          resultMessage = "IP address is found to be clean.";
        }
        else if (ipdata.Result == 1)
        {
          resultMessage = "IP address is found to be malicious.";
        }
        else
        {
          resultMessage = "Could not determine the IP address reputation.";
        }
        return resultMessage;
      },
      render: function() {
        var ipdata = this.props.data.ip;
        if (!isip) {
          return (<div></div>);
        }
        else return (
          <div className="main-section-divider">
            <div>DBot IP Report for:
            <h2>{ipdata.details}</h2>
            </div>
            <h3> {this.resultmessage()} </h3>
            <h3> Risk Score: {ipdata.xfe.ip_reputation.score} </h3>
            <h3> Country: {ipdata.xfe.ip_reputation.geo? ipdata.xfe.ip_reputation.geo['country']:'Unknown'} </h3>

            <div className="panel-group" id="accordion" role="tablist" aria-multiselectable="true">
              <SubnetSection data={ipdata.xfe.ip_reputation.subnets} />
              <ResolutionSection data={ipdata.VT.ip_report.Resolutions}/>
              <DetectedURLSection data={ipdata.VT.ip_report.detected_urls}/>
            </div>
          </div>
        );
      }
    });

    var DetectedURLRow = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        return(
          <tr>
          <td>{urldata.url}</td>
          <td>{urldata.positives}</td>
          <td>{urldata.scan_date}</td>
          </tr>

        );
      }
    });

    var DetectedURLSection = React.createClass({
      render: function() {
        var detected_url_arr = this.props.data;
        var rows = [];
        if (detected_url_arr != null && detected_url_arr.length > 0) {
          for (var i=0; i < detected_url_arr.length; i++) {
            rows.push(<DetectedURLRow urldata={detected_url_arr[i]} />)
          }
          return (

            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="headingThree">
                <h4 className="panel-title">
                  <a className="collapsed" role="button" data-toggle="collapse" data-parent="#accordion" href="#collapseThree" aria-expanded="false" aria-controls="collapseThree">
                    Detected URLs
                  </a>
                </h4>
              </div>
              <div id="collapseThree" className="panel-collapse collapse" role="tabpanel" aria-labelledby="headingThree">
                <div className="panel-body">
                  <div>
                  <table className="table">
                  <thead>
                  <th className="col-lg-8">URL</th>
                  <th className="col-lg-1">Positives</th>
                  <th className="col-lg-3">Scan Date</th>
                  </thead>
                  <tbody>
                  {rows}
                  </tbody>
                  </table>
                  </div>
                </div>
              </div>
            </div>
            );
        }
        else
        {
          return (<div></div>);
        }

      }
    });

    var ResolutionRow = React.createClass({
      render: function() {
        var resdata = this.props.resdata;
        return (
          <tr>
          <td>{resdata.hostname}</td>
          <td>{resdata.last_resolved}</td>
          </tr>
        );
      }
    });

    var ResolutionSection = React.createClass({
      render: function() {
        var resArr = this.props.data;
        var rows = [];
        if (resArr != null && resArr.length > 0) {
          for (var i=0; i < resArr.length; i++) {
            rows.push(<ResolutionRow resdata={resArr[i]} />)
          }
          return (
            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="headingTwo">
                <h4 className="panel-title">
                  <a className="collapsed" role="button" data-toggle="collapse" data-parent="#accordion" href="#collapseTwo" aria-expanded="false" aria-controls="collapseTwo">
                    Historical Resolutions
                  </a>
                </h4>
              </div>
              <div id="collapseTwo" className="panel-collapse collapse" role="tabpanel" aria-labelledby="headingTwo">
                <div className="panel-body">
                  <div>
                    <table className="table">
                      <thead>
                        <th>Hostname</th>
                        <th>Last Resolved</th>
                      </thead>
                      <tbody>
                        {rows}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
            );
        }
        else
          return (<div></div>);

      }
    });

    var SubnetSection = React.createClass({
      render: function() {
        var rows = [];
        var subnets = this.props.data;
        if (subnets != null && subnets.length > 0) {
          for (var i=0; i < subnets.length; i++) {
            rows.push(<SubnetSectionRow subnetdata={subnets[i]} />)
          }
          return (

            <div className="panel panel-default">
              <div className="panel-heading" role="tab" id="headingOne">
                <h4 className="panel-title">
                  <a role="button" data-toggle="collapse" data-parent="#accordion" href="#collapseOne" aria-expanded="true" aria-controls="collapseOne">
                    Subnets
                  </a>
                </h4>
              </div>
              <div id="collapseOne" className="panel-collapse collapse in" role="tabpanel" aria-labelledby="headingOne">
                <div className="panel-body">
                  <div>
                  <table className="table">
                  <thead>
                  <th>Subnet</th>
                  <th>IP</th>
                  <th>Category</th>
                  <th>Location</th>
                  </thead>
                  <tbody>
                  {rows}
                  </tbody>
                  </table>
                  </div>
                </div>
              </div>
            </div>
          );
        } else {
          return (
            <div></div>
          );
        }
      }
    });


    var SubnetSectionRow = React.createClass({
      render: function() {
        var subnetdata = this.props.subnetdata;
        var geodata = "";
        if (subnetdata.geo != null) {
          geodata = subnetdata.geo['country'];
        }
        var category = "";
        var catsMap = subnetdata.cats;
        var keys = Object.keys(catsMap);
        var numEntries = keys.length;

        for (var i=0; i < numEntries; i++) {
          category += keys[i];
          if (i != numEntries-1) {
            category += ", ";
          }
        }
        return (
          <tr>
          <td>{subnetdata.subnet}</td>
          <td>{subnetdata.ip}</td>
          <td>{category}</td>
          <td>{geodata}</td>
          </tr>
        );
      }
    });

    // URL Details Section
    var UrlDiv = React.createClass({
      resultmessage: function() {
        var resultMessage;
        var urldata = this.props.data.url;
        if (urldata.Result == 0)
        {
          resultMessage = "URL is found to be clean.";
        }
        else if (urldata.Result == 1)
        {
          resultMessage = "URL is found to be malicious.";
        }
        else
        {
          resultMessage = "Could not determine the URL reputation.";
        }
        return resultMessage;
      },

      render: function() {
        var urldata = this.props.data.url;
        if (!isurl) {
          return (<div></div>);
        } else return (
          <div className="main-section-divider">
          DBot URL Report for:
          <h2>{urldata.details}</h2>
          <h3> {this.resultmessage()} </h3>
          <h3> Risk Score: {urldata.xfe.url_details.score} </h3>
          <DetectedEngines urldata={urldata} />
          <ARecord urldata={urldata} />
          <AAAARecord urldata={urldata} />
          <TXTRecord urldata={urldata} />
          <MXRecord urldata={urldata} />
          <URLCategory urldata={urldata} />

          </div>
        );
      }
    });

    var ARecord = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var arecord = "";
        if (urldata.xfe.resolve.A != null && urldata.xfe.resolve.A.length > 0) {
          for (var i=0; i < urldata.xfe.resolve.A.length; i++) {
            arecord = arecord + urldata.xfe.resolve.A[i] + " ";
          }
          return(
            <div>
              <h3>A Record </h3>
              {arecord}
            </div>
          );
        } else {
          return(
            <div />
          );
        }
      }
    });

    var AAAARecord = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var aaaarecord = "";
        if (urldata.xfe.resolve.AAAA != null && urldata.xfe.resolve.AAAA.length > 0) {
          for (var i=0; i < urldata.xfe.resolve.AAAA.length; i++) {
            aaaarecord = aaaarecord + urldata.xfe.resolve.AAAA[i] + " ";
          }
          return(
            <div>
              <h3>AAAA Record </h3>
              {aaaarecord}
            </div>
          );
        } else {
          return(
            <div />
          );
        }
      }
    });

    var TXTRecord = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var txtrecord = "";
        if (urldata.xfe.resolve.TXT != null && urldata.xfe.resolve.TXT.length > 0) {
          for (var i=0; i < urldata.xfe.resolve.TXT.length; i++) {
            txtrecord = txtrecord + urldata.xfe.resolve.TXT[i] + " ";
          }
          return(
            <div>
              <h3>TXT Record </h3>
              {txtrecord}
            </div>
          );
        } else {
          return(
            <div />
          );
        }
      }
    });

    var MXRecord = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var mxrecord = "";
        var rows =[];
        if (urldata.xfe.resolve.MX != null && urldata.xfe.resolve.MX.length > 0) {
          for (var i=0; i < urldata.xfe.resolve.MX.length; i++) {
            rows.push(<MXRecordRow mxdata={urldata.xfe.resolve.MX[i]} />);
          }

          return(
            <div>
              <h3>MX Record </h3>
              <table className="table">
                <thead>
                  <th>Exchange</th>
                  <th>Priority</th>
                </thead>
                <tbody>
                  {rows}
                </tbody>
              </table>
            </div>
          );
        } else {
          return(
            <div />
          );
        }
      }
    });

    var MXRecordRow = React.createClass({
        render: function() {
          var mxdata = this.props.mxdata;
          return (
            <tr>
            <td>{mxdata.exchange}</td>
            <td>{mxdata.priority}</td>
            </tr>
          );
        }
    });

    var URLCategory = React.createClass({
      render: function() {
        var urldata=this.props.urldata;
        var cats = urldata.xfe.url_details.cats;
        var catfound = false;
        var category = "";
        for (var k in cats) {
          if (cats[k]) {
            catfound = true;
            // TODO what about multiple categories?
            category = k;
            return (
              <div><h3>Category: {category}</h3></div>
            );
          }
        }
        return (<div/>);
      }
    });

    var DetectedEngines = React.createClass({
      render: function() {
        var urldata = this.props.urldata;
        var detected_engines = " ";
        // display the engines that convicted this URL
        if (urldata.vt.url_report.positives > 0) {
          var numEngines = urldata.vt.url_report.scans.length;
          var scanMap = urldata.vt.url_report.scans;
          for (var k in scanMap) {
            if (scanMap[k].detected) {
              detected_engines = detected_engines + k + " ";
            }
          }
          return(
            <div>
              <h3> Detected Engines : </h3>
              <h4> {detected_engines} </h4>

            </div>
          );
        }
        else {
          return(
            <div>
            </div>
          );

        }
      }
    });


    var FileDiv = React.createClass({
      resultmessage: function() {
        var resultMessage;
        var filedata = null;
        var md5data = null;
        if (isfile) {
          filedata = this.props.data.file;
          md5data = this.props.data.md5;
          if (filedata.Result == 0)
          {
            resultMessage = "File is found to be clean.";
          }
          else if (filedata.Result == 1)
          {
            resultMessage = "File is found to be malicious.";
          }
          else
          {
            resultMessage = "Could not determine the File reputation.";
          }
        }
        else {
          md5data = this.props.data.md5;
          if (md5data.Result == 0)
          {
            resultMessage = "File is found to be clean.";
          }
          else if (md5data.Result == 1)
          {
            resultMessage = "File is found to be malicious.";
          }
          else
          {
            resultMessage = "Could not determine the File reputation.";
          }
        }
        return resultMessage;
      },

      render: function() {
        var data = this.props.data;
        if (!isfile && !ismd5) {
          return (<div></div>);
        } else return (
          <div className="main-section-divider">
          <FileNameHeader data={this.props.data} />
          <h3> {this.resultmessage()} </h3>
          <FileResult filedata={this.props.data.file} />
          <MD5Result md5data={this.props.data.md5} />
          </div>
        );
      }
    });

    var FileNameHeader = React.createClass({
      render: function() {
        var data = this.props.data;
        if (isfile)
        {
          return (
            <div>
              DBot File Report for:
              <h2>{data.file.details.name}</h2>
            </div>
          );
        }
        else if (ismd5) {
          return (
            <div>
              DBot File Report for:
              <h2>{data.md5.details}</h2>
            </div>
          );
        }
      }
    });

    var FileResult = React.createClass({
      render: function() {
        var filedata = this.props.filedata;
        var outstring = "";
        if (filedata.virus) {
          return (
            <div>
              <h2> Malware Name: </h2> {filedata.virus}
            </div>
          );
        }
        else return null;
      }
    });

    var MD5Result = React.createClass ({
      render: function() {
        var md5data = this.props.md5data;
        var numVTDetections = md5data.vt.file_report.positives;
        var scan_row;
        var malware_family_string = "";
        if (md5data.xfe.malware.family) {
          malware_family_string = "Malware Family: " + md5data.xfe.malware.family;
        }
        var detection_string = "";
        if (numVTDetections > 0) {
          detection_string = "Positive Detections: " + numVTDetections;

        }

        return (
          <div>
          {malware_family_string}
          {detection_string}
          <ScanResult data={md5data.vt.file_report} />
          </div>
        );
      }
    });

    var ScanResult = React.createClass ({
      render: function() {
        var file_reports_map = this.props.data.scans;
        var rows = [];
        for (var k in file_reports_map) {
          rows.push(<ScanResultRow enginename={k} scandata={file_reports_map[k]} />);
        }

        return (
          <div>
          <table className="table">
          <thead>
          <th>Engine Name</th>
          <th>Version</th>
          <th>Detected</th>
          <th>Result</th>
          <th>Update</th>
          </thead>
          <tbody>
          {rows}
          </tbody>
          </table>
          </div>
          );

      }

    });

    var ScanResultRow = React.createClass ({
      render: function() {
        var enginename = this.props.enginename;
        var scandata = this.props.scandata;
        return (
          <tr>
          <td>{enginename}</td>
          <td>{scandata.version}</td>
          <td>{scandata.detected?"True":"False"}</td>
          <td>{scandata.result}</td>
          <td>{scandata.update}</td>
          </tr>
        );
      }
    });

    var DetailsDiv = React.createClass({
      loadDataFromServer: function() {
        $.ajax({
          type: 'GET',
          url: '/work',
          data: qParts,
          headers: {'X-XSRF-TOKEN': Cookies.get('XSRF')},
          dataType: 'json',
          contentType: 'application/json; charset=utf-8',
          success: function(data) {
            isfile = (data.type & FILEMask)  > 0;
            ismd5 = (data.type & MD5Mask)  > 0;
            isurl = (data.type & URLMask)  > 0;
            isip = (data.type & IPMask)  > 0;
            this.setState({data: data});
            this.setState({status: 1});
          }.bind(this),
          error: function(xhr, status, error) {
            // display the error on the UI
            this.setState({status: 2});
            var err = error;
            if (xhr && xhr.responseJSON && xhr.responseJSON.errors && xhr.responseJSON.errors[0]) {
              err += " - " + xhr.responseJSON.errors[0].detail;
              this.setState({errmsg: err});
            }
          }.bind(this)
        });
      },
      getInitialState: function() {
        return {data: [], status: 0};
      },
      componentDidMount: function() {
        this.loadDataFromServer();
      },
      render: function() {
            if (this.state.status == 0) {
              return(
                <div>
                  <h2><center>DBot is collecting security details for your query. It might take upto a minute!</center></h2>
                  <br/>
                  <br/>
                  <div className="ball-grid-pulse center-block">
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                    <div></div>
                  </div>

                </div>
              );
            } else if (this.state.status == 1) {
              return (
                <div>
                  <center><h1>DBot Analysis Report</h1></center>
                  <IpDiv data={this.state.data} />
                  <hr></hr>
                  <UrlDiv data={this.state.data} />
                  <hr></hr>
                  <FileDiv data={this.state.data} />
                  <hr></hr>
                </div>
              );
            }
            else {
              return(
                <div>
                DBot encountered an error while trying to serve your request. The issues has been reported and will be analyze.
                Please try to click the link again from Slack interface.
                <hr></hr>
                {this.state.errmsg}
                </div>
              );
            }
        }
  });

  //


  React.render(
    <DetailsDiv />,
    document.getElementById('detailsdiv')
  );

  }

    })(window.jQuery);

// END Settings Handler
// -----------------------------------
