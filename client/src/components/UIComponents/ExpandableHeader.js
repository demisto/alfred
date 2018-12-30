import './ExpandableHeader.less';
import React from 'react';
import classNames from 'classnames';

export function ExpandableHeader({ title, expand ,onClick}) {
  const iconClass = classNames('chevron', { down: expand, right: !expand } , 'icon');

  return (
    <div className="expandable-header" onClick={onClick}>
      <i className={iconClass}/>
      {title}
    </div>
  );
}
