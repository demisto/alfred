
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
        return (
          <div>
            <h1>Details Page for {ipdata.details}</h1>
            <h2> {this.resultmessage()} </h2>
            <h2> Risk Score: {ipdata.xfe.ip_reputation.score} </h2>
            <h2> Country: {ipdata.xfe.ip_reputation.geo['country']} </h2>

            <div className="panel-group" id="accordion" role="tablist" aria-multiselectable="true">
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
                  <SubnetSection data={ipdata.xfe.ip_reputation.subnets} />
                  </div>
                </div>
              </div>
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
                  <ResolutionSection data={ipdata.VT.ip_report.Resolutions}/>
                </div>
              </div>
            </div>
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
                  <DetectedURLSection data={ipdata.VT.ip_report.detected_urls}/>
                </div>
              </div>
            </div>
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
        }
        return (
          <div>
          <table className="table">
          <thead>
          <th>URL</th>
          <th>Positives</th>
          <th>Scan Date</th>
          </thead>
          <tbody>
          {rows}
          </tbody>
          </table>
          </div>
          );

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
        }
        return (
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
          );

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
        }
        return (
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
        );
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
        return (
          <div>
          <h1>Details Page for {urldata.details}</h1>
          <h2> {this.resultmessage()}
           Risk Score: {urldata.xfe.url_details.score} </h2>
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
        if (urldata.xfe.resolve.MX != null && urldata.xfe.resolve.MX.length > 0) {
          for (var i=0; i < urldata.xfe.resolve.MX.length; i++) {
            mxrecord = mxrecord + urldata.xfe.resolve.MX[i] + " ";
          }
          //TODO : MXrecord is an object
          return(
            <div>
              <h3>MX Record </h3>
              {mxrecord}
            </div>
          );
        } else {
          return(
            <div />
          );
        }
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
              <div><h3>{category}</h3></div>
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
//      var filemalicious = false;
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
  //          filemalicious = true;
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
            filemalicious = true;
          }
          else
          {
            resultMessage = "Could not determine the File reputation.";
          }
        }


        return resultMessage;
      },

    render: function() {
//        if (isfile) {
          // this is indeed a file case

//        } else {
          // has to be md5 and not a file.

//        }
        var data = this.props.data;
        return (
          <div>
          <h1>Details Page for {data.file.details.name}</h1>
          </div>
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
          }.bind(this)
        });
      },
      getInitialState: function() {
        return {data: []};
      },
      componentDidMount: function() {
        this.loadDataFromServer();
      },
      render: function() {
        if (isip) {
          return (
          <div>
          <h1>Details Page</h1>
            <IpDiv data={this.state.data} />
          </div>
        );
      }
      else if (isurl) {
        return (
          <div>
          <UrlDiv data={this.state.data} />
          </div>
        );
      }
      else if (isfile || ismd5) {
        return (
          <div>
          <FileDiv data={this.state.data} />
          </div>);
      }
      else return(<div></div>);
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
