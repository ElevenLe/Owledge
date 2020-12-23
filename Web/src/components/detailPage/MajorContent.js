import React, {Component} from "react"
import { Avatar, Card } from 'antd';
import head from "../../assets/user.png"
import { EditOutlined, EllipsisOutlined, SettingOutlined } from '@ant-design/icons';

const {Meta} = Card

class MajorContent extends Component{
    render(){
        return(
            <div>
            <Card style={{height: 500}}
                 >
                <Meta
                      title={this.props.nodeContent.Title}
                      description={this.props.nodeContent.Content}
                      />
            </Card>
            </div>
        )
    }
}

export default MajorContent