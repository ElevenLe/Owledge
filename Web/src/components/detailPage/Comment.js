import React, {Component} from "react"
import {Form, Button, Input, Comment, Tooltip, Avatar, Empty} from 'antd';
import avatar from "../../assets/user.png"
import PersonalComment from "./PersonalComment";
import {API_ROOT} from "../constants"
import CommentList from "./CommentList";


const { TextArea } = Input


const Editor = ({onChange, onSumbit, submitting, value}) => (
    <div>
        <Form.Item>
            <TextArea rows={4} onChange={onChange} value={value}></TextArea>
        </Form.Item>
        <Form.Item>
            <Button htmlType="submit" loading={submitting} onClick={onSumbit} type="primary">Add Comment</Button>
        </Form.Item>
    </div>
)

class NodeComment extends Component{
    state = {
        submitting: false,
        value: '',
    }

    handleSubmit = () => {
        if(!this.state.value){
            return
        }

        this.setState({
            submitting:true
        })

        let id = Math.floor(Math.random()*100 + 3)
        let newPost = {
            commentID: `${this.props.nodeId}c${id}`,
            NodeID: this.props.nodeId,
            AuthorID: "user3",
            Content: this.state.value,
        }

        let postData = JSON.stringify(newPost)        
        fetch(`${API_ROOT}/${this.props.nodeId}` ,{
            method:"POST",
            body: postData,
        }).then((response) => {
            if (response.ok) {
                return this.props.loadNodeContent(this.props.nodeId)
            }
            throw new Error("Failed to post new Comments into backend")
        }).then(() => {
            this.setState({
                submitting: true
            })
        }).catch(err => {
            console.log("post err", err)
            this.setState({
                submitting: false
            })
        })
    }

    deleteComment = (commentId) => {
        let deleteInfo = {
            commentID:commentId,
            AuthorID: "user3",
        }
        let postData = JSON.stringify(deleteInfo)
        fetch(`${API_ROOT}/${this.props.nodeId}/delete`, {
            method: "POST",
            body: postData,
        }).then((response) => {
            if(response.ok){
                return this.props.loadNodeContent(this.props.nodeId)
            }
            throw new Error("Failed to delete the comments")
        }).catch(err => {
            console.log("delete err", err)
        })
    }
    
    updateComment = (commentId, updatePost) => {
        let postData = ""
        if(updatePost != null || updatePost != "null"){
            postData = JSON.stringify(updatePost)
        }
        fetch(`${API_ROOT}/${this.props.nodeId}/${commentId}/update`, {
            method: "POST",
            body: postData
        }).then((response) => {
            if(response.ok){
                return this.props.loadNodeContent(this.props.nodeId)
            }
            throw new Error("Failed to update the comments")
        }).catch(err => {
            console.log("update err", err)
        })
    }

    

    handleChange = e => {
        this.setState({
            value: e.target.value,
        })
    }

    renderComments = () =>{
        const {nodeComments} = this.props
        if(nodeComments.length == null){
            return(
                <Empty description={false}></Empty>
            )
        }
        else{
            let newComments = []
            for(let i =0; i<nodeComments.length; i++){
                let oneComment = {Id: "", AuthorID: "", Content: "", Avatar: avatar}
                oneComment.Id = nodeComments[i].commentID
                oneComment.AuthorID = nodeComments[i].AuthorID
                oneComment.Content = nodeComments[i].Content
                newComments.push(oneComment)
            }
            return <CommentList updateComment={this.updateComment} deleteComment={this.deleteComment} nodeId={this.props.nodeId} newComments={newComments}></CommentList>
        }
    }

    renderCheck = () => {
       if(this.props.nodeComments == null) {
           return <Empty description={false} />
       }
       else {
           return this.renderComments()
       }
    }

    
    render(){
        const {value} = this.state
        return(
            <div>
            {this.renderCheck()}
            <div>
            <Comment
                avatar={<Avatar src={avatar} alt="user3"></Avatar>}
                content={
                    <Editor
                        onChange={this.handleChange}
                        onSumbit={this.handleSubmit}
                        submitting={false}
                        value={value} />
                }
                >
            </Comment>
            </div>
            <div>
            </div>
            </div>
        )
    }
}

export default NodeComment