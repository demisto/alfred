import './ReputationHeader.less'
import { REPUTATION_RESULT } from '../../utils/constants';
import classNames from 'classnames';
import React from 'react';

export function ReputationHeader({ indicatorName, isPrivate, result }) {
  let headerMessage = `Could not determine the ${indicatorName} address reputation.`;
  let headerClass = 'unknown';

  if (isPrivate) {
    headerMessage = `${indicatorName} address is a private (internal) ${indicatorName} - no reputation found.`;
    headerClass = 'clean';
  } else if (result === REPUTATION_RESULT.clean) {
    headerMessage = `${indicatorName} address is found to be clean.`;
    headerClass = 'clean';
  } else if (result === REPUTATION_RESULT.dirty) {
    headerMessage = `${indicatorName} address is found to be malicious.`;
    headerClass = 'dirty';
  }

  return (
    <div className={classNames('ui segment reputation-header', headerClass)}>
      <h3>
        {headerMessage}
      </h3>
    </div>
  );
}
