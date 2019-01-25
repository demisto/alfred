import './ReputationHeader.less'
import { REPUTATION_RESULT } from '../../utils/constants';
import classNames from 'classnames';
import React from 'react';

export function ReputationHeader({ indicatorName, isPrivate, result }) {
  let headerMessage = `Could not determine the ${indicatorName} reputation.`;
  let headerClass = 'unknown';

  if (isPrivate) {
    headerMessage = `${indicatorName} is a private (internal).`;
    headerClass = 'clean';
  } else if (result === REPUTATION_RESULT.clean) {
    headerMessage = `${indicatorName} is found to be clean.`;
    headerClass = 'clean';
  } else if (result === REPUTATION_RESULT.dirty) {
    headerMessage = `${indicatorName} is found to be malicious.`;
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
