import './IPDetails.less';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import autobind from 'autobind-decorator';
import { compareDate, dateToString, keysToString } from '../../utils/utils';
import { flatten, sortBy } from 'lodash';
import Table from '../UIComponents/Table';
import { ReputationHeader } from './ReputationHeader';
import { ExpandableHeader } from '../UIComponents/ExpandableHeader';
import { HeaderSection } from './HeaderSection';

function foundXFE(xfe) {
  const { notFound, resolve } = xfe || {};
  return !notFound || resolve && resolve.A;
}

class URLDetails extends Component {
  static propTypes = {
    details: PropTypes.string,
    result: PropTypes.number,
    xfe: PropTypes.object,
    vt: PropTypes.object
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

    const { resolve, urlDetails, urlMalware } = xfe;

    const headers = [{
      label: 'Risk Score:',
      value: urlDetails && urlDetails.score || 'Unknown'
    }, {
      label: 'Categories:',
      value: keysToString(urlDetails && urlDetails.cats)
    }];

    const { A, AAAA, TXT, MX } = resolve || {};

    const tableData = [
      {
        name: 'A Records',
        value: (A || []).join(', '),
      },
      {
        name: 'AAAA Records',
        value: (AAAA || []).join(', '),
      },
      {
        name: 'TXT Records',
        value: flatten(TXT || []).join(', '),
      },
      {
        name: 'TXT Records',
        value: (MX || []).map(({ exchange, priority }) => `${exchange} (${priority})`).join(', '),
      }
    ];

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
                  title="Records"
                  data={tableData}
                  keys={['name', 'value']}
                  style={{ width: '80%' }}
                />
                <Table
                  title="Malware detected on URL"
                  data={(urlMalware.malware || []).sort(compareDate('firstseen')).map(m => ({
                    ...m,
                    family: (m.family || []).join(', ') || 'Unknown',
                    firstseen: dateToString(m.firstseen),
                  }))}
                  keys={['firstseen', 'type', 'md5', 'uri', 'family']}
                  headers={['First Seen', 'Type', 'MD5', 'URI', 'Family']}
                />
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

    const { urlReport } = vt;

    const headers = [{
      label: 'Scan Date:',
      value: urlReport && urlReport.scan_date || 'Unknown'
    }, {
      label: 'Positives:',
      value: urlReport && urlReport.positives || 0
    }, {
      label: 'Total:',
      value: urlReport && urlReport.total || 0
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
                title="Positive Detections"
                data={Object.entries(urlReport && urlReport.scans || {})
                  .filter(([engine, scan]) => scan.detected)
                  .map(([engine, scan]) => ({
                    ...scan,
                    engine,
                  }))}
                keys={['engine', 'result']}
                headers={['Scan Engine', 'Result']}
                style={{ width: '50%' }}
              />
            </div>
          </div>
        }
      </div>
    )
  }



  render() {
    const { details, result } = this.props;
    return (
      <div className="url-details">
        <h2>URL: {details}</h2>
        <ReputationHeader
          indicatorName="URL"
          result={result}
        />
        {this.getXFESection()}
        {this.getVTSection()}
      </div>
    );
  }
}


export default URLDetails;
