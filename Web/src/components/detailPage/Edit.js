import React, {Component} from "react"
import {Comment, Form, List, Icon, Button, Input, Drawer} from 'antd';

const {TextArea} = Input

const Editor = ({onChange, onSumbit, submitting, value}) => (
    <div>
        <Form.Item>
            <TextArea rows={4} onChange={onChange} value={value}></TextArea>
        </Form.Item>
        <Form.Item>
            <Button htmlType="submit" loading={submitting} onClick={onSumbit} type="primary">Update Comment</Button>
        </Form.Item>
    </div>
)

class Edit extends Component{
    state = {
        visible: false,
        confirmLoading: false,
        value: '',
    }

    showDrawer = () => {
        console.log("edit button click")
        this.setState({
            visible: true,
        })
    }

    onCloseDrawer = () => {
        this.setState({
            visible: false
        })
    }

    handleChange = e => {
        this.setState({
            value: e.target.value
        })
    }

    handleUpdateSubmit = () => {
        if(!this.state.value){
            return
        }

        let updatePost = {
            Summary: "",
            Content: this.state.value
        }

        this.props.updateComment(this.props.commentId,updatePost)
        this.onCloseDrawer()
    }

    updateCommentArea = () => {
        const {value} = this.state
        return (
            <div>
            <Drawer 
                title="update your comment"
                width={500}
                onClose={this.onCloseDrawer}
                visible={this.state.visible}
                bodyStyle={{paddingBottom: 80}}>
                <Editor
                    onChange={this.handleChange}
                    onSumbit={this.handleUpdateSubmit}
                    submitting={false}
                    value={value}>
                </Editor>
            </Drawer>
            </div> 
        )
    }

    
    handleDeleteButton = () => {
        this.props.deleteComment(this.props.commentId)
    }

    handleEditButton = () => {
        this.showDrawer()
    }
    
    checkNodeUser = () =>{
        if (this.props.author=="user3"){
            return (
                <div>
                <div>
                <Button icon="edit" onClick={this.showDrawer}>Edit</Button>
                {this.updateCommentArea()}
                </div>
                <div>
                <Button icon="delete" onClick={this.handleDeleteButton}>Delete</Button>
                </div>
                </div>
            )
        }else {
            return (
                <div></div>
            )
        }
    }
    render(){
        return(
            <div>
            {this.checkNodeUser()}
            </div>
        )
    }
}

export default Edit