import './FileDetails.less';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import autobind from 'autobind-decorator';
import { compareDate } from '../../utils/utils';
import { flatten, sortBy } from 'lodash';
import Table from '../UIComponents/Table';
import { ReputationHeader } from './ReputationHeader';
import { ExpandableHeader } from '../UIComponents/ExpandableHeader';
import { HeaderSection } from './HeaderSection';

function foundXFE(xfe) {
  return xfe && !xfe.notFound;
}

class FileDetails extends Component {
  static propTypes = {
    details: PropTypes.string,
    result: PropTypes.number,
    xfe: PropTypes.object,
    vt: PropTypes.object,
    file: PropTypes.object
  };

  constructor(props) {
    super(props);

    this.state = {
      expandXFE: true,
      expandVT: !foundXFE(props.xfe)
    };
  }

  @autobind
  onToggleXFE() {
    this.setState({ expandXFE: !this.state.expandXFE });
  }

  @autobind
  onToggleVT() {
    this.setState({ expandVT: !this.state.expandVT });
  }

  getXFESection() {
    const { xfe } = this.props;
    const { expandXFE } = this.state;
    if (!foundXFE(xfe)) {
      return null;
    }

    const { type, mimetype, md5, family, created, origins } = xfe.malware || {};

    const headers = [{
      label: 'Type:',
      value: type
    }, {
      label: 'Mime Type:',
      value: mimetype
    }, {
      label: 'MD5:',
      value: md5
    }, {
      label: 'Family:',
      value: (family || []).join(', ') || 'Unknown'
    }, {
      label: 'Created:',
      value: created
    }];

    const { emails, subjects, downloadServers, CnCServers, externals } = origins || {};

    return (
      <div className="ui left aligned grid">
        <div className="row">
          <ExpandableHeader
            title="IBM X-Force Exchange Data"
            expand={expandXFE}
            onClick={this.onToggleXFE}
          />
        </div>
        {
          expandXFE &&
            <div className="row ui left aligned padded grid">
              <HeaderSection
                headers={headers}
              />
              <div className="row">
                <Table
                  title="Email Origins"
                  data={(emails && emails.rows || []).sort(compareDate('firstseen'))}
                  keys={['firstseen', 'lastseen', 'origin', 'md5', 'filepath']}
                  headers={['First Seen', 'Last Seen', 'Origin', 'MD5', 'File Path']}
                />
                <Table
                  title="Subjects"
                  data={(subjects && subjects.rows || [])
                    .sort(compareDate('firstseen'))
                    .map( s => ({ ...s, ips: s.ips && s.ips.join(', ')}))
                  }
                  keys={['firstseen', 'lastseen', 'subject', 'ips',]}
                  headers={['First Seen', 'Last Seen', 'Subject', 'IPs']}
                />
                <Table
                  title="Download Servers"
                  data={(downloadServers && downloadServers.rows || []).sort(compareDate('firstseen'))}
                  keys={['firstseen', 'lastseen', 'host', 'uri',]}
                  headers={['First Seen', 'Last Seen', 'Host', 'URI']}
                />
                <Table
                  title="Commands & Control Servers"
                  data={(CnCServers && CnCServers.rows || [])
                    .sort(compareDate('firstseen'))
                    .map( s => ({ ...s, family: s.family && s.family.join(', ')}))
                  }
                  keys={['firstseen', 'lastseen', 'ip', 'family',]}
                  headers={['First Seen', 'Last Seen', 'IP', 'Family']}
                />
                {
                  externals && externals.family && externals.family.length > 0 &&
                    <div className="external-detection">
                      <div className="h4">External Detection</div>
                      <div className="h5">{externals.family.join(',')}</div>
                    </div>
                }
              </div>
            </div>
        }
      </div>
    )
  }

  getVTSection() {
    const { vt } = this.props;
    const { expandVT } = this.state;

    if (!vt) {
      return null;
    }

    const { fileReport } = vt;

    const headers = [{
      label: 'Scan Date:',
      value: fileReport && fileReport.scan_date || 'Unknown'
    }, {
      label: 'Positives:',
      value: fileReport && fileReport.positives || 0
    }, {
      label: 'Total:',
      value: fileReport && fileReport.total || 0
    }, {
      label: 'MD5:',
      value: fileReport && fileReport.md5 || 'N/A'
    }, {
      label: 'SHA1:',
      value: fileReport && fileReport.sha1 || 'N/A'
    }, {
      label: 'SHA256:',
      value: fileReport && fileReport.sha256 || 'N/A'
    }];

    return (
      <div className="ui left aligned grid">
        <div className="row">
          <ExpandableHeader
            title="Virus Total Data"
            expand={expandVT}
            onClick={this.onToggleVT}
          />
        </div>
        {
          expandVT &&
          <div className="row ui left aligned padded grid">
            <HeaderSection
              headers={headers}
            />
            <div className="row">
              <Table
                title="Detection Engines"
                data={Object.entries(fileReport && fileReport.scans || {})
                  .filter(([engine, scan]) => scan.detected)
                  .map(([engine, scan]) => ({
                    ...scan,
                    engine,
                  }))}
                keys={['engine', 'version', 'result', 'update']}
                headers={['Engine Name', 'Version','Result', 'Update']}
                style={{ width: '50%' }}
              />
            </div>
          </div>
        }
      </div>
    )
  }



  render() {
    const { details, result, file } = this.props;
    let readyDetails = details;
    let readyResult = result;
    if (file) {
      readyDetails = file.details && file.details.name;
      readyResult = file.result;
    }
    return (
      <div className="file-details">
        <div className="h4">File: {readyDetails}</div>
        <ReputationHeader
          indicatorName="File"
          result={readyResult}
        />
        {
          file && file.virus &&
            <div className="malware-name h5">Malware Name: {file.virus}</div>
        }
        {this.getXFESection()}
        {this.getVTSection()}
      </div>
    );
  }
}


export default FileDetails;
