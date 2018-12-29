import './IPDetails.css';
import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { REPUTATION_RESULT } from '../../utils/constants';
import autobind from 'autobind-decorator';
import classNames from 'classnames';
import { keysToString } from '../../utils/utils';
import sortBy from 'lodash/sortBy';

class IPDetails extends Component {
  static propTypes = {
    details: PropTypes.string,
    result: PropTypes.number,
    isPrivate: PropTypes.bool,
    xfe: PropTypes.object,
    vt: PropTypes.object
  };

  constructor(props) {
    super(props);

    this.state = {
      expandXFE: false,
      expandVT: false
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

  getHeader(isPrivate, result) {
    let headerMessage = 'Could not determine the IP address reputation.';
    let headerClass = 'unknown-reputation';

    if (isPrivate) {
      headerMessage = 'IP address is a private (internal) IP - no reputation found.';
      headerClass = 'clean-reputation';
    } else if (result === REPUTATION_RESULT.clean) {
      headerMessage = 'IP address is found to be clean.';
      headerClass = 'clean-reputation';
    } else if (result === REPUTATION_RESULT.dirty) {
      headerMessage = 'IP address is found to be malicious.';
      headerClass = 'dirty-reputation';
    }

    return (
      <h3 className={headerClass}>
        {headerMessage}
      </h3>
    );
  }

  @autobind
  getXFE() {
    const { xfe } = this.props;
    const { expandXFE } = this.state;
    const { notFound, error, ipReputation, ipHistory } = xfe || { notFound: true };
    if (notFound) {
      return null;
    }

    if (error) {
      return (
        <div className="error-message">
          {error}
        </div>
      );
    }

    const iconClass = classNames('chevron', { down: expandXFE, right: !expandXFE } , 'icon');

    return (
      <div>
        <a onClick={this.onToggleXFE}>
          <i className={iconClass}/> IBM X-Force Exchange Data
        </a>
        {
          expandXFE &&
            <div className="xfe-container">
              <h3> Risk Score: {ipReputation.score}</h3>
              <h3> Country: {ipReputation.geo && ipReputation.geo.country || 'Unknown'} </h3>
              <h3> Categories: {keysToString(ipReputation.cats)} </h3>
              {
                ipReputation.subnets && ipReputation.subnets.length > 0 &&
                <div>
                  <h4>Subnets</h4>
                  <table className="ui celled table">
                    <thead>
                      <th>Subnet</th>
                      <th>Score</th>
                      <th>Category</th>
                      <th>Location</th>
                      <th>Reason</th>
                      <th>Created</th>
                    </thead>
                    <tbody>
                      {sortBy(ipReputation.subnets, 'created')
                        .map(({ subnet, score, cats, geo, reason, created }) => (
                        <tr key={subnet}>
                          <td>{subnet}</td>
                          <td>{score}</td>
                          <td>{keysToString(cats)}</td>
                          <td>{geo && geo.country || 'Unknown'}</td>
                          <td>{reason}</td>
                          <td>{created}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              }
              {
                ipHistory && ipHistory.history && ipHistory.history.length > 0 &&
                <div>
                  <h4>IP History</h4>
                  <table className="ui striped table">
                    <thead>
                      <th>IP</th>
                      <th>Score</th>
                      <th>Category</th>
                      <th>Location</th>
                      <th>Reason</th>
                      <th>Created</th>
                    </thead>
                    <tbody>
                    {sortBy(ipHistory.history, 'created')
                      .map(({ ip, score, cats, geo, reason, created }) => (
                        <tr key={ip}>
                          <td>{ip}</td>
                          <td>{score}</td>
                          <td>{keysToString(cats)}</td>
                          <td>{geo && geo.country || 'Unknown'}</td>
                          <td>{reason}</td>
                          <td>{created}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              }
            </div>
        }
      </div>
    )
  }



  render() {
    const { details, result, isPrivate } = this.props;
    const header = this.getHeader(isPrivate, result);
    return (
      <div>
        <h2>IP: {details}</h2>
        {header}
        {this.getXFE()}
      </div>
    );
  }
}


export default IPDetails;
