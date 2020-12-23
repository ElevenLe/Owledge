import React, {Component} from "react"
import logo from "../assets/owledgeDesign.svg"
import {Icon} from "antd";

class TopBar extends Component {
    render() {
        return (
            <header className = "App-header">
                <img src={logo} alt="logo" className="App-logo"/>
            </header>
        )
    }
}

export default TopBar