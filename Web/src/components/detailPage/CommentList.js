import React, {Component} from "react"
import {Comment, Form, List, Icon, Button, Input, Drawer} from 'antd';
import avatar from "../../assets/user.png"
import {API_ROOT} from "../constants"
import Edit from "./Edit"

class CommentList extends Component{
    render(){
        return(
            <div>
            <List 
                className="comment-list"
                header={`${this.props.newComments.length} commends`}
                itemLayout="horizontal"
                dataSource={this.props.newComments}
                renderItem={item => (
                        <li>
                            <Comment
                                author={item.AuthorID}
                                avatar={avatar}
                                content={item.Content}
                            />
                            <Edit updateComment={this.props.updateComment} deleteComment={this.props.deleteComment} author={item.AuthorID} commentId={item.Id}/>
                            {console.log(item.Id)}
                        </li>
                )}
            >
            </List>
            </div>
        )
    }
}

export default CommentList