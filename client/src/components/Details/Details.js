import './Details.less';
import React, { Component } from 'react';
import { API_RESPONSE_STATUS } from '../../utils/constants';
import { parseType } from '../../utils/utils';
import { get } from '../../utils/api';
import IPDetails from "./IPDetails";
import URLDetails from './URLDetails';
import FileDetails from './FileDetails';

class Details extends Component {

  constructor(props) {
    super(props);

    this.state = {
      status: '',
      loading: false,
      data: {}
    };
  }

  componentDidMount() {
      const url = `/work${window.location.search}`;
      this.setState({ loading: true }, async () => {
        const { status, data } = await get(url);
        this.setState({ status, data, loading: false })
      });
  }

  getDetailsSection(data) {
    const { type, ips, urls, file, hashes  } = data;
    const { isIP, isURL, isFile, isMD5 } = parseType(type);
    return (
      <div className="ui centered grid">
        <div className="row">
          <h3 className="text-center">D<small>BOT</small> Analysis Report:</h3>
        </div>
        <div className="row">
          {
            isIP && ips && ips[0] && <IPDetails {...ips[0]}/>
          }
          {
            isURL && urls && urls[0] && <URLDetails {...urls[0]}/>
          }
          {
            (isMD5 || isFile) && hashes && hashes[0] && <FileDetails {...hashes[0]} file={isFile ? file : null}  />
          }
          {
            !type && <div className="h5"> Could not find any result. Sorry... </div>
          }
        </div>
      </div>
    );
  }

  render() {
    const { status, loading, data } = this.state;
    return (
      <div className="details-page">
        {loading &&
          <div className="ui centered grid loading-wrapper">
            <div className="row">
              <div className="h2">D<small>BOT</small> is collecting security details for your query. It might take up to a minute!</div>
            </div>
            <div className="row">
              <div className="sk-cube-grid">
                <div className="sk-cube sk-cube1"></div>
                <div className="sk-cube sk-cube2"></div>
                <div className="sk-cube sk-cube3"></div>
                <div className="sk-cube sk-cube4"></div>
                <div className="sk-cube sk-cube5"></div>
                <div className="sk-cube sk-cube6"></div>
                <div className="sk-cube sk-cube7"></div>
                <div className="sk-cube sk-cube8"></div>
                <div className="sk-cube sk-cube9"></div>
              </div>
            </div>
          </div>
        }
        {!loading && status === API_RESPONSE_STATUS.error &&
          <div className="ui centered grid">
            <div className="raw">
              <h1> OOps </h1>
              D<small>BOT</small> encountered an error while trying to serve your request. The issues has been reported and will be analyzed.
              Please try to click the link again from Slack interface.
            </div>
            <div className="row">
              Error details: { JSON.stringify(data, null, 2) }
            </div>
          </div>
        }

        {!loading && status === API_RESPONSE_STATUS.success && this.getDetailsSection(data)}
      </div>
    );
  }
}

export default Details;
